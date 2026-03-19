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

	// Инициализация репозиториев и сервисов для ленты
	likeToPostRepo := repository.NewLikeToPostRepo()
	commentRepo := repository.NewCommentRepo()
	repostRepo := repository.NewRepostRepo()
	postRepo := repository.NewPostRepo()
	profileRepo := repository.NewProfileRepo()
	postService := service.NewPostService(postRepo, profileRepo, likeToPostRepo, commentRepo, repostRepo)

	// инициализация userProfile service
	userRepo := repository.NewUserRepo()
	userProfileRepo := repository.NewUserProfileRepo()
	userProfileService := service.NewUserProfileService(userRepo, profileRepo, userProfileRepo)

	sessionRepo := repository.NewSessionRepo()

	authService := service.NewAuthService(userRepo, profileRepo, userProfileRepo)
	sessService := service.NewSessionService(sessionRepo)

	// _, err := authService.Register(context.Background(), "Kok", "Inside", "KokInside", "hard_password_1", "24/02/2005")
	// if err != nil {
	// 	fmt.Println("Test user already exists or error:", err)
	// }
	authHandler := handlers.NewAuthHandler(authService, sessService, userProfileService)

	mediaRepo := repository.NewMediaRepo()
	postWithMediaRepo := repository.NewPostWithMediaRepo()
	mediaService := service.NewMediaService(mediaRepo, postWithMediaRepo)

	userHandler := &handlers.UserHandler{
		UserService:  userProfileService,
		MediaService: mediaService,
	}
	feedHandler := handlers.NewFeedHandler(postService, mediaService, userProfileService)

	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler)

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
