package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

const userAgent = "ARIS seed-media-minio/1.0"

type mediaRow struct {
	ID        int64
	UID       uuid.UUID
	Name      string
	Extension string
	MimeType  string
	Link      string
}

type downloadedImage struct {
	body      []byte
	mimeType  string
	extension string
}

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithTimeout(context.Background(), envDuration("SEED_MEDIA_TIMEOUT", 10*time.Minute))
	defer cancel()

	db, err := postgres.New(ctx)
	if err != nil {
		log.Fatal("fail to connect PostgreSQL: ", err)
	}
	defer db.Close()

	minioClient, err := xminio.New(ctx, zap.NewNop())
	if err != nil {
		log.Fatal("fail to connect MinIO: ", err)
	}

	bucketName := utils.EnvString("MINIO_BUCKET_NAME", "media")
	items, err := externalMedia(ctx, db)
	if err != nil {
		log.Fatal("fail to load external media links: ", err)
	}
	if len(items) == 0 {
		log.Println("no external media links found")
		return
	}

	client := &http.Client{Timeout: envDuration("SEED_MEDIA_DOWNLOAD_TIMEOUT", 20*time.Second)}
	maxBytes := int64(utils.EnvInt("SEED_MEDIA_MAX_BYTES", 10<<20))

	var uploaded int
	for _, item := range items {
		image, err := downloadImage(ctx, client, item.Link, maxBytes)
		if err != nil {
			log.Fatalf("fail to download media %d (%s): %v", item.ID, item.Link, err)
		}

		objectName := xminio.GenerateMediaName(item.UID, int64(len(image.body)), image.extension)
		uploadInfo, err := minioClient.PutObject(
			ctx,
			bucketName,
			objectName,
			bytes.NewReader(image.body),
			int64(len(image.body)),
			minio.PutObjectOptions{ContentType: image.mimeType},
		)
		if err != nil {
			log.Fatalf("fail to upload media %d to MinIO: %v", item.ID, err)
		}

		newLink := objectPublicPath(uploadInfo.Bucket, uploadInfo.Key)
		if err := updateMediaLink(ctx, db, item, newLink, image); err != nil {
			log.Fatalf("fail to update media %d link: %v", item.ID, err)
		}

		uploaded++
		log.Printf("media %d (%s): %s -> %s", item.ID, item.Name, item.Link, newLink)
	}

	log.Printf("uploaded %d media files to MinIO", uploaded)
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func externalMedia(ctx context.Context, db queryer) ([]mediaRow, error) {
	rows, err := db.Query(ctx, `
		SELECT id, uid, media_name, extension, mime_type, link
		FROM media
		WHERE is_active = TRUE
		  AND link ~* '^https?://'
		  AND (mime_type ILIKE 'image%' OR extension IN ('jpg', 'jpeg', 'png', 'webp', 'gif'))
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []mediaRow
	for rows.Next() {
		var item mediaRow
		if err := rows.Scan(&item.ID, &item.UID, &item.Name, &item.Extension, &item.MimeType, &item.Link); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func downloadImage(ctx context.Context, client *http.Client, sourceURL string, maxBytes int64) (downloadedImage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return downloadedImage{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	if strings.Contains(sourceURL, "rupixel.ru/") {
		req.Header.Set("Referer", "https://www.rupixel.ru/")
	}

	resp, err := client.Do(req)
	if err != nil {
		return downloadedImage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return downloadedImage{}, fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return downloadedImage{}, err
	}
	if int64(len(body)) > maxBytes {
		return downloadedImage{}, fmt.Errorf("image is larger than %d bytes", maxBytes)
	}

	detected := mimetype.Detect(body)
	if !strings.HasPrefix(detected.String(), "image/") {
		return downloadedImage{}, fmt.Errorf("unsupported mime type %s", detected.String())
	}

	return downloadedImage{
		body:      body,
		mimeType:  detected.String(),
		extension: detected.Extension(),
	}, nil
}

func updateMediaLink(ctx context.Context, db execer, item mediaRow, newLink string, image downloadedImage) error {
	tag, err := db.Exec(ctx, `
		UPDATE media
		SET link = $1,
		    size = $2,
		    extension = $3,
		    mime_type = $4,
		    updated_at = NOW()
		WHERE id = $5
		  AND link = $6
	`, newLink, len(image.body), strings.TrimPrefix(image.extension, "."), image.mimeType, item.ID, item.Link)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("media link was changed concurrently")
	}
	return nil
}

func objectPublicPath(bucketName, objectName string) string {
	return "/" + strings.Trim(bucketName, "/") + "/" + strings.TrimLeft(objectName, "/")
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := utils.EnvString(name, "")
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
