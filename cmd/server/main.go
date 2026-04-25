package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-park-mail-ru/2026_1_ARIS/docs"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"

	authhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	chathandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/chat"
	feedhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	friendshiphandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/friend"
	mediahandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/media"
	posthandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/post"
	profilehandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/profile"
	supporthandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/support"
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
	supportrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/support"
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
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/support"
	supportservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/support"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	connectdb "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_db"
	connectminio "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_minio"
	connectredis "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_redis"
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

	// Создаём логгер

	logConf := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			TimeKey:      "time",
			NameKey:      "logger_name",
			CallerKey:    "caller",
			FunctionKey:  "function",
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	logger, err := logConf.Build()
	if err != nil {
		log.Fatalf("Can't configure logger %s", err)
	}
	if logger == nil {
		log.Fatal("logger is nil")
	}
	logger.Info("logger initialized")

	defer func() {
		if err := logger.Sync(); err != nil {
			if errors.Is(err, syscall.EINVAL) {
				return
			}
			logger.Error("Failed to sync logger", zap.Error(err))
		}
	}()

	//sugar := logger.Sugar()

	envConf, err := config.NewConfig()
	if err != nil {
		logger.Fatal("failed to load env variables", zap.Error(err))
	}

	confStr, err := connectdb.GetConnectURL(envConf)
	if err != nil {
		logger.Fatal("failed to get db connection string: ", zap.Error(err))
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, confStr)
	if err != nil {
		logger.Fatal("fail to connect to db: ", zap.Error(err))
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		logger.Fatal("failed db connection check: ", zap.Error(err))
	}

	logger.Info("Successfully connected to PostgreSQL")

	// Подключаем Redis

	redisClient, err := connectredis.InitRedis(ctx, envConf)
	if err != nil {
		logger.Fatal("fail to connect to Redis", zap.Error(err))
	}

	logger.Info("Successfully connected to Redis")

	// Подключаем MinIO

	// создание MinIO клиента
	minioClient, err := connectminio.InitMinio(envConf)
	if err != nil {
		logger.Fatal("fail to initialize MinIO", zap.Error(err))
	}

	// Проверка на существование бакета
	exists, err := minioClient.BucketExists(ctx, envConf.MinioBucketName)
	if err != nil {
		logger.Fatal("fail to chech MinIO bucket existition", zap.Error(err))
	}

	logger.Info("Successfully connected to MinIO")

	// Если бакета нет - его нужно создать
	if !exists {
		err := minioClient.MakeBucket(ctx, envConf.MinioBucketName, minio.MakeBucketOptions{})
		if err != nil {
			logger.Fatal("fail to create MinIO buchet", zap.Error(err))
		}
		logger.Info(("MinIO bucket created"))
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
		logger.Fatal("fail to marshal MinIO policy", zap.Error(err))
	}

	err = minioClient.SetBucketPolicy(ctx, envConf.MinioBucketName, string(rawPolicy))
	if err != nil {
		logger.Fatal("fail to set MinIO bucket policy", zap.Error(err))
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
	sessionRepo := sessionrepo.NewSessionStorage(redisClient)
	mediaRepo := mediarepo.NewMediaStorage(db)
	postWithMediaRepo := postrepo.NewPostWithMediaStorage(db)
	friendshipRepo := friendshiprepo.NewFriendshipStorage(db)
	settingsRepo := settingsrepo.NewUserSettingsStorage(db)
	chatRepo := chatrepo.NewChatStorage(db)
	chatMemberRepo := chatmemberrepo.NewChatMemberStorage(db)
	messageRepo := messagerepo.NewMessageStorage(db)
	supportTicketRepo := supportrepo.NewTicketStorage(db)

	logger.Info("repositories initialized")

	postService := postservice.NewPostService(postRepo, postWithMediaRepo, profileRepo, commentRepo, repostRepo, likeRepo)
	userService := userservice.NewUserService(userAccountRepo, profileRepo, userProfileRepo)
	authService := authservice.NewAuthService(userAccountRepo, profileRepo, userProfileRepo)
	sessService := sessionservice.NewSessionService(sessionRepo)
	mediaService := mediaservice.NewMediaService(mediaRepo, postWithMediaRepo, client)
	chatSvc := chatservice.NewChatService(chatRepo, chatMemberRepo, userService)
	messageSvc := messageservice.NewMessageService(messageRepo)
	friendshipService := friendshipservice.NewFriendshipService(friendshipRepo)
	settingsService := settingsservice.NewUserSettingsService(settingsRepo)
	ticketService := supportservice.NewTicketService(supportTicketRepo, mediaRepo)

	logger.Info("services initialized")

	// WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	logger.Info("ws started")

	userHandler := userhandler.NewUserHandler(userService, mediaService, settingsService)
	authHandler := authhandler.NewAuthHandler(authService, sessService, userService, mediaService)
	authHandler.SetSupportService(ticketService)
	feedHandler := feedhandler.NewFeedHandler(postService, mediaService, userService)
	mediaHandler := mediahandler.NewMediaHandler(mediaService, sessService, userService)
	profileHandler := profilehandler.NewProfileHandler(userService, mediaService, sessService)
	chatHandler := chathandler.NewChatHandler(chatSvc, messageSvc, userAccountRepo, userService, hub)
	friendHandler := friendshiphandler.NewFriendHandler(sessService, userService, friendshipService)
	wsHandler := wsHandler.NewWebSocketHandler(hub, chatSvc)
	postHandler := posthandler.NewPostHandler(userService, postService, mediaService)
	supportHandler := supporthandler.NewSupportHandler(sessService, userService, ticketService, hub)

	logger.Info("handlers initialized")

	utils.MakeMock(mediaRepo, userService, postService, postWithMediaRepo, commentRepo, repostRepo, chatRepo, likeRepo)

	support.MakeProfileAdmin(ctx, ticketService, 1)
	support.MakeProfileAdmin(ctx, ticketService, 2)
	support.MakeProfileAdmin(ctx, ticketService, 3)
	support.MakeProfileAdmin(ctx, ticketService, 4)

	// создаём роутер
	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler, mediaHandler, profileHandler, chatHandler, friendHandler, wsHandler, postHandler, supportHandler, logger)

	cop := http.NewCrossOriginProtection()
	cop.AddTrustedOrigin("http://localhost:3001")
	cop.AddTrustedOrigin("https://localhost:3001")
	cop.AddTrustedOrigin("http://arisnet.ru")
	cop.AddTrustedOrigin("https://arisnet.ru")

	mainMux := http.NewServeMux()

	mainMux.Handle("/ws/", router)
	mainMux.Handle("/", cop.Handler(router))

	// создаём роутер

	fmt.Println("Swagger is running on http://localhost:8080/swagger/index.html")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mainMux,
	}

	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		logger.Info("server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("fail to server listen", zap.Error(err))
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
