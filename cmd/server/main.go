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

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"

	chatrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	commentrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	likerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	mediarepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	postrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	profilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	repostrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
	sessionrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	useraccountrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"

	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/auth"
	mediaservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	postservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	sessionservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"

	authhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	feedhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/profile"
	userhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	connectdb "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// @title						ARIS backend API
// @version						1.0.0
// @description					Description of ARIS backend API
// @host						localhost:8080
// @BasePath					/api
// @accept						json
// @produce						json
// @schemes						http
// @securityDefinitions.apikey	SessionAuth
// @in							cookie
// @name						session_id
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

	// Инициализация репозиториев

	//commentRepo := commentrepo.NewCommentRepo()
	commentRepo := commentrepo.NewCommentStorage(db)

	//repostRepo := repostrepo.NewRepostRepo()
	repostRepo := repostrepo.NewRepostStorage(db)

	//postRepo := postrepo.NewPostRepo()
	postRepo := postrepo.NewPostStorage(db)

	//profileRepo := profilerepo.NewProfileRepo()
	profileRepo := profilerepo.NewProfileStorage(db)

	//likeRepo := likerepo.NewLikeRepo()
	likeRepo := likerepo.NewLikeStorage(db)

	//userRepo := useraccountrepo.NewUserRepo()
	userAccountRepo := useraccountrepo.NewUserAccountStorage(db)

	//userProfileRepo := userprofilerepo.NewUserProfileRepo()
	userProfileRepo := userprofilerepo.NewUserProfileStorage(db)

	sessionRepo := sessionrepo.NewSessionRepo()

	//mediaRepo := mediarepo.NewMediaRepo()
	mediaRepo := mediarepo.NewMediaStorage(db)

	//postWithMediaRepo := postrepo.NewPostWithMediaRepo()
	postWithMediaRepo := postrepo.NewPostWithMediaStorage(db)

	chatRepo := chatrepo.NewChatStorage(db)

	// инициализация сервисов
	postService := postservice.NewPostService(postRepo, profileRepo, commentRepo, repostRepo, likeRepo)
	userService := userservice.NewUserService(userAccountRepo, profileRepo, userProfileRepo)
	authService := authservice.NewAuthService(userAccountRepo, profileRepo, userProfileRepo)
	sessService := sessionservice.NewSessionService(sessionRepo)
	mediaService := mediaservice.NewMediaService(mediaRepo, postWithMediaRepo)

	// инициализация хэндлеров
	authHandler := authhandler.NewAuthHandler(authService, sessService, userService)
	userHandler := &userhandler.UserHandler{
		UserService:  userService,
		MediaService: mediaService,
	}
	feedHandler := feedhandler.NewFeedHandler(postService, mediaService, userService)
	profileHandler := profile.NewProfileHandler(userService, mediaService, sessService)

	// заполнение тестовыми данными
	utils.MakeMock(mediaRepo, userService, postService, postWithMediaRepo, commentRepo, repostRepo, chatRepo)

	// swagger
	fmt.Println("Swagger is running on http://localhost:8080/swagger/index.html")

	// создаём роутер
	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler, profileHandler)

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
