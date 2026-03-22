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

	commentrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	likerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	mediarepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	postrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	profilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	repostrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
	sessionrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	useraccountrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"

	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/auth"
	mediaservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	postservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	sessionservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"

	authhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	feedhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	userhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"
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
	// Инициализация репозиториев
	commentRepo := commentrepo.NewCommentRepo()
	repostRepo := repostrepo.NewRepostRepo()
	postRepo := postrepo.NewPostRepo()
	profileRepo := profilerepo.NewProfileRepo()
	likeRepo := likerepo.NewLikeRepo()
	userRepo := useraccountrepo.NewUserRepo()
	userProfileRepo := userprofilerepo.NewUserProfileRepo()
	sessionRepo := sessionrepo.NewSessionRepo()
	mediaRepo := mediarepo.NewMediaRepo()
	postWithMediaRepo := postrepo.NewPostWithMediaRepo()

	// инициализация сервисов
	postService := postservice.NewPostService(postRepo, profileRepo, commentRepo, repostRepo, likeRepo)
	userService := userservice.NewUserService(userRepo, profileRepo, userProfileRepo)
	authService := authservice.NewAuthService(userRepo, profileRepo, userProfileRepo)
	sessService := sessionservice.NewSessionService(sessionRepo)
	mediaService := mediaservice.NewMediaService(mediaRepo, postWithMediaRepo)

	// инициализация хэндлеров
	authHandler := authhandler.NewAuthHandler(authService, sessService, userService)
	userHandler := &userhandler.UserHandler{
		UserService:  userService,
		MediaService: mediaService,
	}
	feedHandler := feedhandler.NewFeedHandler(postService, mediaService, userService)

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
	utils.MakeMock(mediaRepo, userService, postService, postWithMediaRepo, commentRepo, repostRepo)

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
