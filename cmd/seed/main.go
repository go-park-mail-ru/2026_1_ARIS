package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
	useraccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	envConf, err := config.NewConfig()
	if err != nil {
		log.Fatal("fail to load env variables: ", err)
	}

	db, err := postgres.New(ctx, envConf)
	if err != nil {
		log.Fatal("fail to connect PostgreSQL: ", err)
	}
	defer db.Close()

	minioClient, err := xminio.New(ctx, envConf, zap.NewNop())
	if err != nil {
		log.Fatal("fail to connect MinIO: ", err)
	}

	commentRepo := comment.NewCommentStorage(db)
	repostRepo := repost.NewRepostStorage(db)
	postRepo := post.NewPostStorage(db)
	profileRepo := profile.NewProfileStorage(db)
	likeRepo := like.NewLikeStorage(db)
	userAccountRepo := useraccount.NewUserAccountStorage(db)
	userProfileRepo := userprofile.NewUserProfileStorage(db)
	mediaRepo := media.NewMediaStorage(db)
	postWithMediaRepo := post.NewPostWithMediaStorage(db)
	chatRepo := chat.NewChatStorage(db)

	utils.MakeMock(
		mediaRepo,
		userAccountRepo,
		profileRepo,
		userProfileRepo,
		postRepo,
		postWithMediaRepo,
		commentRepo,
		repostRepo,
		chatRepo,
		likeRepo,
		media.NewMinioClient(minioClient),
		envConf.MinioBucketName,
	)

	log.Println("seed data completed")
}
