package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-park-mail-ru/2026_1_ARIS/docs"
	"go.uber.org/zap"

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

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment: ", err)
	}

	ctx := context.Background()

	// Создаём логгер

	logger, err := logger.New()
	if err != nil {
		log.Fatal("fail to create logger: ", err)
	}
	logger.Info("logger initialized")

	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("fail to sync logger", zap.Error(err))
		}
	}()

	// Создаём конфиг

	envConf, err := config.NewConfig()
	if err != nil {
		logger.Fatal("fail to load env variables", zap.Error(err))
	}
	logger.Info("Config parsed")

	// Подключаем к БД

	db, err := postgres.New(ctx, envConf)
	if err != nil {
		logger.Fatal("fail to connect PostgreSQL", zap.Error(err))
	}
	logger.Info("Successfully connected to PostgreSQL")

	// Подключаем Redis

	redisClient, err := redis.InitRedis(ctx, envConf)
	if err != nil {
		logger.Fatal("fail to connect Redis", zap.Error(err))
	}

	logger.Info("Successfully connected to Redis")

	// Подключаем MinIO

	minioClient, err := xminio.New(ctx, envConf, logger)
	if err != nil {
		logger.Fatal("fail to connect MinIO", zap.Error(err))
	}
	logger.Info("Successfully connected to MinIO")

	S3client := mediarepo.NewMinioClient(minioClient)

	// Инициализация репозиториев

	commentRepo := commentrepo.NewCommentStorage(db)
	repostRepo := repostrepo.NewRepostStorage(db)
	postRepo := postrepo.NewPostStorage(db)
	profileRepo := profilerepo.NewProfileStorage(db)
	likeRepo := likerepo.NewLikeStorage(db)
	userAccountRepo := useraccountrepo.NewUserAccountStorage(db)
	userProfileRepo := userprofilerepo.NewUserProfileStorage(db)
	sessionRepo := sessionrepo.NewSessionStorage(redisClient)
	mediaRepo := mediarepo.NewMediaStorage(db)
	postWithMediaRepo := postrepo.NewPostWithMediaStorage(db)
	friendshipRepo := friendshiprepo.NewFriendshipStorage(db)
	settingsRepo := settingsrepo.NewUserSettingsStorage(db)
	chatRepo := chatrepo.NewChatStorage(db)
	chatMemberRepo := chatmemberrepo.NewChatMemberStorage(db)
	messageRepo := messagerepo.NewMessageStorage(db)

	logger.Info("repositories initialized")

	postService := postservice.NewPostService(postRepo, postWithMediaRepo, profileRepo, commentRepo, repostRepo, likeRepo)
	userService := userservice.NewUserService(userAccountRepo, profileRepo, userProfileRepo)
	authService := authservice.NewAuthService(userAccountRepo, profileRepo, userProfileRepo)
	sessService := sessionservice.NewSessionService(sessionRepo)
	mediaService := mediaservice.NewMediaService(mediaRepo, postWithMediaRepo, S3client)
	chatSvc := chatservice.NewChatService(chatRepo, chatMemberRepo, userService)
	messageSvc := messageservice.NewMessageService(messageRepo)
	friendshipService := friendshipservice.NewFriendshipService(friendshipRepo)
	settingsService := settingsservice.NewUserSettingsService(settingsRepo)

	logger.Info("services initialized")

	// WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	logger.Info("ws started")

	userHandler := userhandler.NewUserHandler(userService, mediaService, settingsService)
	authHandler := authhandler.NewAuthHandler(authService, sessService, userService, mediaService)
	feedHandler := feedhandler.NewFeedHandler(postService, mediaService, userService)
	mediaHandler := mediahandler.NewMediaHandler(mediaService, sessService, userService)
	profileHandler := profilehandler.NewProfileHandler(userService, mediaService, sessService)
	chatHandler := chathandler.NewChatHandler(chatSvc, messageSvc, userAccountRepo, mediaService, userService, hub)
	friendHandler := friendshiphandler.NewFriendHandler(sessService, userService, friendshipService)
	wsHandler := wsHandler.NewWebSocketHandler(hub, chatSvc)
	postHandler := posthandler.NewPostHandler(userService, postService, mediaService)

	logger.Info("handlers initialized")

	// создаём роутер
	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler, mediaHandler, profileHandler, chatHandler, friendHandler, wsHandler, postHandler, logger)

	cop := http.NewCrossOriginProtection()
	cop.AddTrustedOrigin("http://localhost:3001")
	cop.AddTrustedOrigin("https://localhost:3001")
	cop.AddTrustedOrigin("http://arisnet.ru")
	cop.AddTrustedOrigin("https://arisnet.ru")
	cop.AddTrustedOrigin("https://arisnet.online")

	mainMux := http.NewServeMux()

	mainMux.Handle("/ws/", router)
	mainMux.Handle("/", cop.Handler(router))

	fmt.Println("Swagger is running on http://localhost:8080/swagger/index.html")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mainMux,
	}

	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		logger.Info("server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("fail to server listen")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Server is stopping")
	logger.Info("Server is stopping")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Fatal("Server forced to shutdown:", zap.Error(err))
	}

	fmt.Println("Server stopped")
	logger.Info("server stopped")
}
