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
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/server"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

func main() {
	// Инициализация репозиториев и сервисов для ленты
	likeToPostRepo := repository.NewLikeToPostRepo()
	commentRepo := repository.NewCommentRepo()
	postRepo := repository.NewPostRepo()
	profileRepo := repository.NewProfileRepo()
	postService := service.NewPostService(postRepo, profileRepo, likeToPostRepo, commentRepo)

	mediaRepo := repository.NewMediaRepo()
	postWithMediaRepo := repository.NewPostWithMediaRepo()
	mediaService := service.NewMediaService(mediaRepo, postWithMediaRepo)

	feedHandler := handlers.NewFeedHandler(postService, mediaService)

	authService := service.NewAuthService()
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

	// инициализация userProfile service
	userRepo := repository.NewUserRepo()
	userProfileRepo := repository.NewUserProfileRepo()
	userProfileService := service.NewUserProfileService(userRepo, profileRepo, userProfileRepo)

	// create user avatars
	avatar1 := models.NewMedia("avatar 1", "png", "user avatar 1", "image", "media/link/avatars/1.png", 1000, false)
	avatar2 := models.NewMedia("avatar 2", "png", "user avatar 2", "image", "media/link/avatars/2.png", 2000, false)
	avatar3 := models.NewMedia("avatar 3", "png", "user avatar 3", "image", "media/link/avatars/3.png", 3000, false)
	avatar4 := models.NewMedia("avatar 4", "png", "user avatar 4", "image", "media/link/avatars/4.png", 4000, false)
	avatar5 := models.NewMedia("avatar 5", "png", "user avatar 5", "image", "media/link/avatars/5.png", 5000, false)
	mediaRepo.Save(context.Background(), avatar1)
	mediaRepo.Save(context.Background(), avatar2)
	mediaRepo.Save(context.Background(), avatar3)
	mediaRepo.Save(context.Background(), avatar4)
	mediaRepo.Save(context.Background(), avatar5)

	// create Profiles (users)
	user1 := userProfileService.CreateRealUserProfile(context.Background(), "email@gmail.com", "+79990001122", "hard password hash", "username 0???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar1)
	user2 := userProfileService.CreateRealUserProfile(context.Background(), "email1@gmail.com", "+179990001122", "hard password hash", "username 1???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar2)
	user3 := userProfileService.CreateRealUserProfile(context.Background(), "email2@gmail.com", "+279990001122", "hard password hash", "username 2???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar3)
	user4 := userProfileService.CreateRealUserProfile(context.Background(), "email3@gmail.com", "+379990001122", "hard password hash", "username 3???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar4)
	user5 := userProfileService.CreateRealUserProfile(context.Background(), "email4@gmail.com", "+479990001122", "hard password hash", "username 4???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar5)

	// create medias
	media1 := models.NewMedia("Media name 1", "png", "Media description 1", "image", "media/link/media1.png", 10241, false)
	media2 := models.NewMedia("Media name 2", "png", "Media description 2", "image", "media/link/media2.png", 10242, false)
	media3 := models.NewMedia("Media name 3", "png", "Media description 3", "image", "media/link/media3.png", 10243, false)
	media4 := models.NewMedia("Media name 4", "png", "Media description 4", "image", "media/link/media4.png", 10244, false)
	media5 := models.NewMedia("Media name 5", "png", "Media description 5", "image", "media/link/media5.png", 10245, false)
	media6 := models.NewMedia("Media name 6", "png", "Media description 6", "image", "media/link/media6.png", 10246, false)
	mediaRepo.Save(context.Background(), media1)
	mediaRepo.Save(context.Background(), media2)
	mediaRepo.Save(context.Background(), media3)
	mediaRepo.Save(context.Background(), media4)
	mediaRepo.Save(context.Background(), media5)
	mediaRepo.Save(context.Background(), media6)

	// create posts
	post1 := models.NewPost("post text 1", user1, true)
	post2 := models.NewPost("post text 2", user2, true)
	post3 := models.NewPost("post text 3", user3, true)
	post4 := models.NewPost("post text 4", user4, true)
	post5 := models.NewPost("post text 5", user5, true)
	postService.Save(context.Background(), post1)
	postService.Save(context.Background(), post2)
	postService.Save(context.Background(), post3)
	postService.Save(context.Background(), post4)
	postService.Save(context.Background(), post5)

	// connect post with medias to get PostWithMedia
	postWithMediaRepo.Save(post1, media1, 0)
	postWithMediaRepo.Save(post2, media2, 0)
	postWithMediaRepo.Save(post3, media3, 0)
	postWithMediaRepo.Save(post3, media6, 1)
	postWithMediaRepo.Save(post4, media4, 0)
	postWithMediaRepo.Save(post5, media5, 0)

	// create likes & init LikeRepo
	like1 := models.NewLike(user4)
	like2 := models.NewLike(user5)
	like3 := models.NewLike(user1)
	like4 := models.NewLike(user2)
	like5 := models.NewLike(user3)
	like6 := models.NewLike(user3)
	likeRepo := repository.NewLikeRepo()
	likeRepo.Save(like1)
	likeRepo.Save(like2)
	likeRepo.Save(like3)
	likeRepo.Save(like4)
	likeRepo.Save(like5)
	likeRepo.Save(like6)

	// create and save LikeToPosts
	likeToPost1 := models.NewLikeToPost(like1.ID, post1.ID)
	likeToPost2 := models.NewLikeToPost(like2.ID, post2.ID)
	likeToPost3 := models.NewLikeToPost(like3.ID, post3.ID)
	likeToPost4 := models.NewLikeToPost(like4.ID, post4.ID)
	likeToPost5 := models.NewLikeToPost(like5.ID, post5.ID)
	likeToPost6 := models.NewLikeToPost(like6.ID, post2.ID)
	likeToPostRepo.Save(likeToPost1)
	likeToPostRepo.Save(likeToPost2)
	likeToPostRepo.Save(likeToPost3)
	likeToPostRepo.Save(likeToPost4)
	likeToPostRepo.Save(likeToPost5)
	likeToPostRepo.Save(likeToPost6)

	// create comments
	comment1 := models.NewComment("comment 1", post1.ID, nil, nil, user2.ID, false)
	comment2 := models.NewComment("comment 2", post2.ID, nil, nil, user3.ID, false)
	comment3 := models.NewComment("comment 3", post3.ID, nil, nil, user4.ID, false)
	comment4 := models.NewComment("comment 4", post4.ID, nil, nil, user5.ID, false)
	comment5 := models.NewComment("comment 5", post5.ID, nil, nil, user1.ID, false)
	comment6 := models.NewComment("comment 6", post1.ID, nil, nil, user4.ID, false)
	commentRepo.Save(comment1)
	commentRepo.Save(comment2)
	commentRepo.Save(comment3)
	commentRepo.Save(comment4)
	commentRepo.Save(comment5)
	commentRepo.Save(comment6)

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
