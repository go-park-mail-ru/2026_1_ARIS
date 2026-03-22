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
	handlers "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	connectdb "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// @title ARIS backend API
// @version 1.0.0
// @description Description of ARIS backend API
// @host localhost:8080
// @BasePath /api
// @accept json
// @produce json
// @schemes http
// @securityDefinitions.apikey SessionAuth
// @in cookie
// @name session_id

func main() {
	// загружаем переменные окружения из файла или systemd
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment")
	}

	// подключаем БД

	// парсим переменные окружения
	envConf, err := config.NewConfig()
	if err != nil {
		log.Fatal("can't get env config", err)
	}

	// создаём URL подключения к Postgres
	confStr, err := connectdb.GetConnectURL(envConf)
	if err != nil {
		log.Fatal("can't get db connection string", err)
	}

	ctx := context.Background()

	// создаём пул подключений
	db, err := pgxpool.New(ctx, confStr)
	if err != nil {
		log.Fatal("can't connect to db", err)
	}
	defer db.Close()

	// пингуем БД для проверки
	err = db.Ping(ctx)
	if err != nil {
		log.Fatal("Bad db connection", err)
	}

	fmt.Println("Successfully connected to PostgreSQL")

	// создаём репозитории
	profileRepo := repository.NewProfileRepo()
	userRepo := repository.NewUserRepo()
	userProfileRepo := repository.NewUserProfileRepo()
	likeToPostRepo := repository.NewLikeToPostRepo()
	commentRepo := repository.NewCommentRepo()
	repostRepo := repository.NewRepostRepo()
	postRepo := repository.NewPostRepo()
	sessionRepo := repository.NewSessionRepo()
	mediaRepo := repository.NewMediaRepo()
	postWithMediaRepo := repository.NewPostWithMediaRepo()

	// создаём сервисы
	postService := service.NewPostService(postRepo, profileRepo, likeToPostRepo, commentRepo, repostRepo)
	userProfileService := service.NewUserProfileService(userRepo, profileRepo, userProfileRepo)
	authService := service.NewAuthService(userRepo, profileRepo, userProfileRepo)
	sessService := service.NewSessionService(sessionRepo)
	mediaService := service.NewMediaService(mediaRepo, postWithMediaRepo)

	// создаём хэндлеры
	authHandler := handlers.NewAuthHandler(authService, sessService, userProfileService)
	userHandler := &handlers.UserHandler{
		UserService:  userProfileService,
		MediaService: mediaService,
	}
	feedHandler := handlers.NewFeedHandler(postService, mediaService, userProfileService)

	// создаём роутер
	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler)

	// создаём сервер
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// запускаем сервер
	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// заполнение тестовыми данными
	utils.MakeMock(mediaRepo, userProfileService, postService, postWithMediaRepo, likeToPostRepo, commentRepo, repostRepo)

	// gracefull shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Server is stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server stopped")
}
