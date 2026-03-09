package utils

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

func MakeMock(mediaRepo repository.MediaRepo,
	userProfileService service.UserService,
	postService service.PostService,
	postWithMediaRepo repository.PostWithMediaRepo,
	likeToPostRepo repository.LikeToPostRepo,
	commentRepo repository.CommentRepo) {

	// create user avatars
	userAvatar1 := "user avatar 1"
	userAvatar2 := "user avatar 2"
	userAvatar3 := "user avatar 3"
	userAvatar4 := "user avatar 4"
	userAvatar5 := "user avatar 5"
	avatar1 := models.NewMedia("avatar 1", "png", &userAvatar1, "image", "media/link/avatars/1.png", 1000, false)
	avatar2 := models.NewMedia("avatar 2", "png", &userAvatar2, "image", "media/link/avatars/2.png", 2000, false)
	avatar3 := models.NewMedia("avatar 3", "png", &userAvatar3, "image", "media/link/avatars/3.png", 3000, false)
	avatar4 := models.NewMedia("avatar 4", "png", &userAvatar4, "image", "media/link/avatars/4.png", 4000, false)
	avatar5 := models.NewMedia("avatar 5", "png", &userAvatar5, "image", "media/link/avatars/5.png", 5000, false)
	mediaRepo.Save(context.Background(), avatar1)
	mediaRepo.Save(context.Background(), avatar2)
	mediaRepo.Save(context.Background(), avatar3)
	mediaRepo.Save(context.Background(), avatar4)
	mediaRepo.Save(context.Background(), avatar5)

	// create Profiles (users)
	user1, _ := userProfileService.CreateRealUserProfile(context.Background(), "email@gmail.com", "+79990001122", "hard password hash", "username 0???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar1)
	user2, _ := userProfileService.CreateRealUserProfile(context.Background(), "email1@gmail.com", "+179990001122", "hard password hash", "username 1???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar2)
	user3, _ := userProfileService.CreateRealUserProfile(context.Background(), "email2@gmail.com", "+279990001122", "hard password hash", "username 2???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar3)
	user4, _ := userProfileService.CreateRealUserProfile(context.Background(), "email3@gmail.com", "+379990001122", "hard password hash", "username 3???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar4)
	user5, _ := userProfileService.CreateRealUserProfile(context.Background(), "email4@gmail.com", "+479990001122", "hard password hash", "username 4???", "cool first name", "not so cool last name", true, nil, models.Gender(1), &avatar5)

	// create medias
	mediaDesctiption1 := "Media description 1"
	mediaDesctiption2 := "Media description 2"
	mediaDesctiption3 := "Media description 3"
	mediaDesctiption4 := "Media description 4"
	mediaDesctiption5 := "Media description 5"
	mediaDesctiption6 := "Media description 6"
	media1 := models.NewMedia("Media name 1", "png", &mediaDesctiption1, "image", "media/link/media1.png", 10241, false)
	media2 := models.NewMedia("Media name 2", "png", &mediaDesctiption2, "image", "media/link/media2.png", 10242, false)
	media3 := models.NewMedia("Media name 3", "png", &mediaDesctiption3, "image", "media/link/media3.png", 10243, false)
	media4 := models.NewMedia("Media name 4", "png", &mediaDesctiption4, "image", "media/link/media4.png", 10244, false)
	media5 := models.NewMedia("Media name 5", "png", &mediaDesctiption5, "image", "media/link/media5.png", 10245, false)
	media6 := models.NewMedia("Media name 6", "png", &mediaDesctiption6, "image", "media/link/media6.png", 10246, false)
	mediaRepo.Save(context.Background(), media1)
	mediaRepo.Save(context.Background(), media2)
	mediaRepo.Save(context.Background(), media3)
	mediaRepo.Save(context.Background(), media4)
	mediaRepo.Save(context.Background(), media5)
	mediaRepo.Save(context.Background(), media6)

	// create posts
	postText1 := "post text 1"
	postText2 := "post text 2"
	postText3 := "post text 3"
	postText4 := "post text 4"
	postText5 := "post text 5"
	post1 := models.NewPost(&postText1, *user1, true)
	post2 := models.NewPost(&postText2, *user2, true)
	post3 := models.NewPost(&postText3, *user3, true)
	post4 := models.NewPost(&postText4, *user4, true)
	post5 := models.NewPost(&postText5, *user5, true)
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
	like1 := models.NewLike(*user4)
	like2 := models.NewLike(*user5)
	like3 := models.NewLike(*user1)
	like4 := models.NewLike(*user2)
	like5 := models.NewLike(*user3)
	like6 := models.NewLike(*user3)
	like7 := models.NewLike(*user1)
	like8 := models.NewLike(*user4)
	likeRepo := repository.NewLikeRepo()
	likeRepo.Save(like1)
	likeRepo.Save(like2)
	likeRepo.Save(like3)
	likeRepo.Save(like4)
	likeRepo.Save(like5)
	likeRepo.Save(like6)
	likeRepo.Save(like7)
	likeRepo.Save(like8)

	// create and save LikeToPosts
	likeToPost1 := models.NewLikeToPost(like1.ID, post1.ID)
	likeToPost2 := models.NewLikeToPost(like2.ID, post2.ID)
	likeToPost3 := models.NewLikeToPost(like3.ID, post3.ID)
	likeToPost4 := models.NewLikeToPost(like4.ID, post4.ID)
	likeToPost5 := models.NewLikeToPost(like5.ID, post5.ID)
	likeToPost6 := models.NewLikeToPost(like6.ID, post2.ID)
	likeToPost7 := models.NewLikeToPost(like7.ID, post2.ID)
	likeToPost8 := models.NewLikeToPost(like8.ID, post2.ID)
	likeToPostRepo.Save(likeToPost1)
	likeToPostRepo.Save(likeToPost2)
	likeToPostRepo.Save(likeToPost3)
	likeToPostRepo.Save(likeToPost4)
	likeToPostRepo.Save(likeToPost5)
	likeToPostRepo.Save(likeToPost6)
	likeToPostRepo.Save(likeToPost7)
	likeToPostRepo.Save(likeToPost8)

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
}
