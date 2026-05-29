package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/chat"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/comment"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/like"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/repost"
	useraccount "github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/user_account"
	userprofile "github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/user_profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/utils"
	globalutils "github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
		globalutils.EnvString("MINIO_BUCKET_NAME", "media"),
	)

	if err := utils.MakeDemoData(ctx, db, media.NewMinioClient(minioClient), globalutils.EnvString("MINIO_BUCKET_NAME", "media")); err != nil {
		log.Fatal("fail to create demo seed data: ", err)
	}
	if err := utils.SeedDemoStickers(ctx, db, mediaRepo, media.NewMinioClient(minioClient), globalutils.EnvString("MINIO_BUCKET_NAME", "media")); err != nil {
		log.Fatal("fail to create demo stickers: ", err)
	}

	log.Println("seed data completed")
}
