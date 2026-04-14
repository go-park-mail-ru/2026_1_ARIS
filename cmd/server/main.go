package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-park-mail-ru/2026_1_ARIS/docs"
	"github.com/minio/minio-go/v7"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"

	authhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	chathandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/chat"
	feedhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	friendshiphandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/friend"
	mediahandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/media"
	posthandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/post"
	profilehandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/profile"
	userhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"
	wsHandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/websocket"

	chatrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	chatmemberrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat_member"
	commentrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	friendshiprepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend"
	likerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	mediarepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	messagerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/message"
	postrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	profilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	repostrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
	sessionrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	settingsrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings"
	useraccountrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"

	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/auth"
	chatservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/chat"
	friendshipservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/friend"
	mediaservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	messageservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/message"
	postservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	sessionservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	settingsservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/settings"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	connectdb "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_db"
	connectminio "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_minio"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment: ", err)
	}

	envConf, err := config.NewConfig()
	if err != nil {
		log.Fatal("Can't get env config:", err)
	}

	confStr, err := connectdb.GetConnectURL(envConf)
	if err != nil {
		log.Fatal("Can't get db connection string: ", err)
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, confStr)
	if err != nil {
		log.Fatal("Can't connect to db: ", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		log.Fatal("Bad db connection: ", err)
	}

	fmt.Println("Successfully connected to PostgreSQL")

	// Подключаем MinIO

	// создание MinIO клиента
	minioClient, err := connectminio.InitMinio(envConf)
	if err != nil {
		log.Fatalf("Ошибка инициализации Minio: %v", err)
	}

	// Проверка на существование бакета
	exists, err := minioClient.BucketExists(ctx, envConf.MinioBucketName)
	if err != nil {
		log.Fatalf("Can't chech bucket existition: %v", err)
	}

	fmt.Println("Successfully connected to MinIO")

	// Если бакета нет - его нужно создать
	if !exists {
		err := minioClient.MakeBucket(ctx, envConf.MinioBucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("Can't create buchet: %v", err)
		}
		fmt.Println("Bucket created")
	}

	// Устанавливаем политику доступа к файлам
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []any{
			map[string]any{
				"Effect": "Allow",
				"Principal": map[string]string{
					"AWS": "*",
				},
				"Action": []string{
					"s3:GetBucketLocation",
					"s3:ListBucket",
				},
				"Resource": "arn:aws:s3:::" + envConf.MinioBucketName,
			},
			map[string]any{
				"Effect": "Allow",
				"Principal": map[string]string{
					"AWS": "*",
				},
				"Action":   "s3:GetObject",
				"Resource": "arn:aws:s3:::" + envConf.MinioBucketName + "/*",
			},
		},
	}

	rawPolicy, err := json.Marshal(policy)
	if err != nil {
		log.Fatalf("Can't marshal policy: %v", err)
	}

	err = minioClient.SetBucketPolicy(ctx, envConf.MinioBucketName, string(rawPolicy))
	if err != nil {
		log.Fatalf("Can't set bucket policy: %v", err)
	}

	client := mediarepo.NewMinioClient(minioClient)

	// Инициализация репозиториев

	commentRepo := commentrepo.NewCommentStorage(db)
	repostRepo := repostrepo.NewRepostStorage(db)
	postRepo := postrepo.NewPostStorage(db)
	profileRepo := profilerepo.NewProfileStorage(db)
	likeRepo := likerepo.NewLikeStorage(db)
	userAccountRepo := useraccountrepo.NewUserAccountStorage(db)
	userProfileRepo := userprofilerepo.NewUserProfileStorage(db)
	sessionRepo := sessionrepo.NewSessionRepo()
	mediaRepo := mediarepo.NewMediaStorage(db)
	postWithMediaRepo := postrepo.NewPostWithMediaStorage(db)
	friendshipRepo := friendshiprepo.NewFriendshipStorage(db)
	settingsRepo := settingsrepo.NewUserSettingsStorage(db)
	chatRepo := chatrepo.NewChatStorage(db)
	chatMemberRepo := chatmemberrepo.NewChatMemberStorage(db)
	messageRepo := messagerepo.NewMessageStorage(db)

	postService := postservice.NewPostService(postRepo, postWithMediaRepo, profileRepo, commentRepo, repostRepo, likeRepo)
	userService := userservice.NewUserService(userAccountRepo, profileRepo, userProfileRepo)
	authService := authservice.NewAuthService(userAccountRepo, profileRepo, userProfileRepo)
	sessService := sessionservice.NewSessionService(sessionRepo)
	mediaService := mediaservice.NewMediaService(mediaRepo, postWithMediaRepo, client)
	chatSvc := chatservice.NewChatService(chatRepo, chatMemberRepo, userService)
	messageSvc := messageservice.NewMessageService(messageRepo)
	friendshipService := friendshipservice.NewFriendshipService(friendshipRepo)
	settingsService := settingsservice.NewUserSettingsService(settingsRepo)

	// WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	userHandler := userhandler.NewUserHandler(userService, mediaService, settingsService)
	authHandler := authhandler.NewAuthHandler(authService, sessService, userService, mediaService)
	feedHandler := feedhandler.NewFeedHandler(postService, mediaService, userService)
	mediaHandler := mediahandler.NewMediaHandler(mediaService, sessService, userService)
	profileHandler := profilehandler.NewProfileHandler(userService, mediaService, sessService)
	chatHandler := chathandler.NewChatHandler(chatSvc, messageSvc, userAccountRepo, userService, hub)
	friendHandler := friendshiphandler.NewFriendHandler(sessService, userService, friendshipService)
	wsHandler := wsHandler.NewWebSocketHandler(hub, chatSvc)
	postHandler := posthandler.NewPostHandler(userService, postService, mediaService)

	utils.MakeMock(mediaRepo, userService, postService, postWithMediaRepo, commentRepo, repostRepo, chatRepo, likeRepo)

	// создаём роутер
	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler, mediaHandler, profileHandler, chatHandler, friendHandler, wsHandler, postHandler)

	ensureKnownTestUser(ctx, userAccountRepo, userService, "sergeyshulginenko", "chatcheck123", "Сергей", "Шульгиненко")
	ensureKnownPassword(ctx, db, "sergeyshulginenko", "chatcheck123")

	fmt.Println("Swagger is running on http://localhost:8080/swagger/index.html")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Server is stopping")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server stopped")
}

func ensureKnownPassword(ctx context.Context, db *pgxpool.Pool, username string, password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	_, _ = db.Exec(ctx, "UPDATE user_account SET password_hash=$1 WHERE username=$2", string(hash), username)
}

func ensureKnownTestUser(
	ctx context.Context,
	userAccountRepo useraccountrepo.UserAccountRepo,
	userService userservice.UserService,
	username string,
	password string,
	firstName string,
	lastName string,
) {
	if _, err := userAccountRepo.GetByUsername(ctx, username); err == nil {
		return
	}
	birthdayDate, err := time.Parse("02/01/2006", "24/02/2005")
	if err != nil {
		return
	}
	_, _ = userService.CreateRealUserProfile(
		ctx,
		nil,
		nil,
		password,
		username,
		firstName,
		lastName,
		birthdayDate,
		models.Gender("male"),
		nil,
	)
}
