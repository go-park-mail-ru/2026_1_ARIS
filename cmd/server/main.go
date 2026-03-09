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

	handlers "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

func main() {
	// Инициализация репозиториев и сервисов для ленты
	likeToPostRepo := repository.NewLikeToPostRepo()
	commentRepo := repository.NewCommentRepo()
	postRepo := repository.NewPostRepo()
	profileRepo := repository.NewProfileRepo()
	postService := service.NewPostService(postRepo, profileRepo, likeToPostRepo, commentRepo)

	// инициализация userProfile service
	userRepo := repository.NewUserRepo()
	userProfileRepo := repository.NewUserProfileRepo()
	userProfileService := service.NewUserProfileService(userRepo, profileRepo, userProfileRepo)

	mediaRepo := repository.NewMediaRepo()
	postWithMediaRepo := repository.NewPostWithMediaRepo()
	mediaService := service.NewMediaService(mediaRepo, postWithMediaRepo)

	feedHandler := handlers.NewFeedHandler(postService, mediaService)

	authService := service.NewAuthService(userRepo)
	jwtSecret := "ключ"
	authHandler := handlers.NewAuthHandler(authService, jwtSecret)

	router := server.NewRouter(authHandler, feedHandler, []byte(jwtSecret))

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
	utils.MakeMock(mediaRepo, userProfileService, postService, postWithMediaRepo, likeToPostRepo, commentRepo)

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
