package repository

import (
	legacymedia "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
)

type Store struct {
	Media      legacymedia.MediaRepo
	S3         legacymedia.S3Repo
	BucketName string
}

func NewStore(
	media legacymedia.MediaRepo,
	s3 legacymedia.S3Repo,
	bucketName string,
) Store {
	return Store{
		Media:      media,
		S3:         s3,
		BucketName: bucketName,
	}
}
