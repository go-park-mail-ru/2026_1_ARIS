package utils

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type demoSticker struct {
	Title     string
	Name      string
	SourceURL string
	Fallback  string
}

func SeedDemoStickers(ctx context.Context, db *pgxpool.Pool, mediaRepo media.MediaRepo, s3Repo media.S3Repo, bucketName string) error {
	if db == nil || mediaRepo == nil || s3Repo == nil || bucketName == "" {
		return nil
	}

	var authorID int64
	if err := db.QueryRow(ctx, `
		SELECT up.profile_id
		FROM user_profile up
		JOIN user_account ua ON ua.id=up.user_account_id
		WHERE ua.username='demoowner'
	`).Scan(&authorID); err != nil {
		return fmt.Errorf("find sticker seed author: %w", err)
	}

	packID, err := ensureStickerPack(ctx, db, "ARIS demo stickers", authorID)
	if err != nil {
		return err
	}

	stickers := []demoSticker{
		{Title: "like", Name: "demo-sticker-like", SourceURL: "https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f44d.svg", Fallback: "👍"},
		{Title: "love", Name: "demo-sticker-love", SourceURL: "https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/2764.svg", Fallback: "❤️"},
		{Title: "laugh", Name: "demo-sticker-laugh", SourceURL: "https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f602.svg", Fallback: "😂"},
		{Title: "sad", Name: "demo-sticker-sad", SourceURL: "https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f622.svg", Fallback: "😢"},
		{Title: "angry", Name: "demo-sticker-angry", SourceURL: "https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg/1f621.svg", Fallback: "😡"},
	}

	for index, sticker := range stickers {
		mediaID, err := ensureStickerMedia(ctx, mediaRepo, s3Repo, bucketName, sticker, authorID)
		if err != nil {
			return err
		}
		if err := ensureSticker(ctx, db, packID, mediaID, index); err != nil {
			return err
		}
	}

	return nil
}

func ensureStickerPack(ctx context.Context, db *pgxpool.Pool, title string, authorID int64) (int64, error) {
	var id int64
	if err := db.QueryRow(ctx, `SELECT id FROM sticker_pack WHERE title=$1 AND is_active=TRUE LIMIT 1`, title).Scan(&id); err == nil {
		return id, nil
	}

	err := db.QueryRow(ctx, `
		INSERT INTO sticker_pack (uid, title, author_id, is_active)
		VALUES ($1, $2, $3, TRUE)
		RETURNING id
	`, uuid.New(), title, authorID).Scan(&id)
	return id, err
}

func ensureStickerMedia(ctx context.Context, mediaRepo media.MediaRepo, s3Repo media.S3Repo, bucketName string, sticker demoSticker, authorID int64) (int64, error) {
	if id, err := mediaRepo.GetIDByName(ctx, sticker.Name); err == nil {
		return id, nil
	}

	description := "Demo sticker: " + sticker.Title
	seedMedia, err := newSeedMediaFromURL(ctx, s3Repo, bucketName, sticker.Name, &description, sticker.SourceURL, authorID)
	if err != nil {
		seedMedia, err = newFallbackStickerMedia(ctx, s3Repo, bucketName, sticker, &description, authorID)
		if err != nil {
			return 0, err
		}
	}

	id, err := mediaRepo.Save(ctx, *seedMedia)
	if err == nil {
		return id, nil
	}
	return mediaRepo.GetIDByName(ctx, sticker.Name)
}

func newFallbackStickerMedia(ctx context.Context, s3Repo media.S3Repo, bucketName string, sticker demoSticker, description *string, authorID int64) (*models.Media, error) {
	body := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256"><rect width="256" height="256" rx="48" fill="#f5f7fb"/><text x="128" y="154" text-anchor="middle" font-family="Arial, sans-serif" font-size="112">%s</text></svg>`, sticker.Fallback))
	detected := mimetype.Detect(body)
	mediaUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("aris-demo-sticker:"+sticker.Name))
	link, err := s3Repo.Save(ctx, bucketName, bytes.NewReader(body), mediaUUID, int64(len(body)), detected.Extension(), minio.PutObjectOptions{
		ContentType: detected.String(),
	})
	if err != nil {
		return nil, err
	}
	return models.NewMedia(sticker.Name, strings.TrimPrefix(detected.Extension(), "."), mediaUUID, description, detected.String(), link, authorID), nil
}

func ensureSticker(ctx context.Context, db *pgxpool.Pool, packID, mediaID int64, order int) error {
	_, err := db.Exec(ctx, `
		INSERT INTO sticker (uid, size, sort_order, pack_id, media_id, is_active)
		VALUES ($1, 0, $2, $3, $4, TRUE)
		ON CONFLICT (pack_id, sort_order) DO UPDATE
		SET media_id=EXCLUDED.media_id, is_active=TRUE, updated_at=NOW()
	`, uuid.New(), order, packID, mediaID)
	return err
}
