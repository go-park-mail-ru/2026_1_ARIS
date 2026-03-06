package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

func main() {
	feedRepo := repository.NewFeedRepo()
	feedService := service.NewFeedService(feedRepo)

	postRepo := repository.NewPostRepo()
	profileRepo := repository.NewProfileRepo()
	postService := service.NewPostService(postRepo, profileRepo)

	mediaRepo := repository.NewMediaRepo()
	postWithMediaRepo := repository.NewPostWithMediaRepo()
	mediaService := service.NewMediaService(mediaRepo, postWithMediaRepo)

	media := models.NewMedia("Media name", "png", "Media description", "image", "media/link/media33223.png", 1024, false)

	mediaRepo.Save(context.Background(), media)

	feedHandler := handler.NewHandler(feedService, postService, mediaService)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello world")
	})

	mux.HandleFunc("/api/feed/", feedHandler.GetFeed)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		fmt.Println("Server started")
		fmt.Println("Server is running on http://localhost:8080")

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Println("Internal server error", err)
			return
		}
	}()

	// инициализация репозитория и добавление тестовых данных
	db := repository.NewRepository()

	KokInside := models.NewUser("KokInside@gmail.com", "+79999999999", "hard_password")

	db.UserRepo.Save(context.Background(), KokInside)

	userProfileRepo := repository.NewUserProfileRepo()

	KokInsideProfile := models.NewProfile("KokInside", nil, true)
	KokInsideUserProfile := models.NewUserProfile(KokInside, KokInsideProfile, "Ivan", "Khvostov", nil, models.Male)

	userProfileRepo.Save(KokInsideUserProfile)

	profileRepo.Save(KokInsideProfile)

	//postService.PostRepo.Save(context.Background(), models.NewPost("Текст поста", KokInsideProfile, true))

	fmt.Println("Посты = ", postRepo.Posts)

	users, err := db.UserRepo.List(context.Background(), 0, 1)
	if err != nil {
		fmt.Println("Error listing users:", err)
	} else {
		fmt.Println("Users:", users)
	}

	// for i := range 10 {
	// 	post := models.Post{
	// 		Text:      fmt.Sprintf("%s_%s", "Post №", strconv.Itoa(i)),
	// 		CreatedAt: time.Now(),
	// 	}
	// 	feedService.Save(context.Background(), post)
	// }

	post := models.NewPost("Текст поста", KokInsideProfile, true)

	postRepo.Save(context.Background(), post)

	postWithMediaRepo.Save(post, media, 0)

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	<-stop

	fmt.Println("Server is stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Server stopped")
}
