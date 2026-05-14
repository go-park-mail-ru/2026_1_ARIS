package repository

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

var ErrMediaNotFound = errors.New("media not found")

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct {
	Media      MediaRepo
	S3         S3Repo
	BucketName string
}

func NewStore(media MediaRepo, s3 S3Repo, bucketName string) Store {
	return Store{Media: media, S3: s3, BucketName: bucketName}
}

type MediaRepo interface {
	Get(ctx context.Context, id int64) (*model.Media, error)
	Save(ctx context.Context, media model.Media) (int64, error)
	GetLink(ctx context.Context, id int64) (string, error)
	UpdateLink(ctx context.Context, id int64, newLink string) error
}

type mediaStorage struct {
	db DB
}

func NewMediaStorage(db DB) MediaRepo {
	return &mediaStorage{db: db}
}

func (s *mediaStorage) Get(ctx context.Context, id int64) (*model.Media, error) {
	var media model.Media
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &media, `SELECT * FROM media WHERE id=$1`, id)
	logQuery(ctx, "mediaStorage.Get", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrMediaNotFound
		}
		return nil, err
	}
	return &media, nil
}

func (s *mediaStorage) Save(ctx context.Context, media model.Media) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `
		INSERT INTO media (uid, media_name, extension, mime_type, size, link, author_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, media.Uid, media.Name, media.Extension, media.MimeType, media.Size, media.Link, media.AuthorID)
	logQuery(ctx, "mediaStorage.Save", start)

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *mediaStorage) GetLink(ctx context.Context, id int64) (string, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `SELECT link FROM media WHERE id=$1`, id)
	logQuery(ctx, "mediaStorage.GetLink", start)

	var link string
	if err := row.Scan(&link); err != nil {
		return "", err
	}
	return link, nil
}

func (s *mediaStorage) UpdateLink(ctx context.Context, id int64, newLink string) error {
	start := time.Now()
	res, err := s.db.Exec(ctx, `UPDATE media SET link=$1 WHERE id=$2`, newLink, id)
	logQuery(ctx, "mediaStorage.UpdateLink", start)
	if err != nil {
		return err
	}
	if res.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}
	return nil
}

type S3Repo interface {
	Save(ctx context.Context, bucketName string, reader io.Reader, mediaUUID uuid.UUID, size int64, extension string, opts minio.PutObjectOptions) (string, error)
}

type minioStorage struct {
	client *minio.Client
}

func NewMinioStorage(client *minio.Client) S3Repo {
	return &minioStorage{client: client}
}

func (s *minioStorage) Save(ctx context.Context, bucketName string, reader io.Reader, mediaUUID uuid.UUID, size int64, extension string, opts minio.PutObjectOptions) (string, error) {
	objectName := xminio.GenerateMediaName(mediaUUID, size, extension)

	start := time.Now()
	uploadInfo, err := s.client.PutObject(ctx, bucketName, objectName, reader, size, opts)
	logMinioPut(ctx, bucketName, objectName, start)
	if err != nil {
		return "", err
	}
	return objectPublicPath(uploadInfo.Bucket, uploadInfo.Key), nil
}

func objectPublicPath(bucketName, objectName string) string {
	return "/" + strings.Trim(bucketName, "/") + "/" + strings.TrimLeft(objectName, "/")
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}

func logMinioPut(ctx context.Context, bucket, object string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("minio put object", zap.String("bucket", bucket), zap.String("object", object), zap.Duration("duration_ms", time.Since(start)))
}
