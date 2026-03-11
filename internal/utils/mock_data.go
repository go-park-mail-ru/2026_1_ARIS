package utils

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/google/uuid"
)

func MakeMock(mediaRepo repository.MediaRepo,
	userProfileService service.UserService,
	postService service.PostService,
	postWithMediaRepo repository.PostWithMediaRepo,
	likeToPostRepo repository.LikeToPostRepo,
	commentRepo repository.CommentRepo,
	repostRepo repository.RepostRepo) {

	// create user avatars
	userAvatar1 := "user avatar 1 description"
	userAvatar2 := "user avatar 2 description"
	userAvatar3 := "user avatar 3 description"
	userAvatar4 := "user avatar 4 description"
	userAvatar5 := "user avatar 5 description"
	avatar1 := models.NewMedia("avatar_1_name", "jpg", &userAvatar1, "image", "https://img02.rl0.ru/afisha/e750x-i/daily.afisha.ru/uploads/images/6/e5/6e5a713fb8d534791c6eed2e47be9640.jpg", 1000, false)
	avatar2 := models.NewMedia("avatar_2_name", "jpg", &userAvatar2, "image", "https://avatarko.ru/img/kartinka/1/Crazy_Frog.jpg", 2000, false)
	avatar3 := models.NewMedia("avatar_3_name", "jpg", &userAvatar3, "image", "https://avatarko.ru/img/kartinka/1/multfilm_pingviny.jpg", 3000, false)
	avatar4 := models.NewMedia("avatar_4_name", "jpg", &userAvatar4, "image", "https://avatarko.ru/img/kartinka/2/kot_uzhasy_1770.jpg", 4000, false)
	avatar5 := models.NewMedia("avatar_5_name", "jpg", &userAvatar5, "image", "https://spbcult.ru/upload/iblock/7b9/9n0tc4etzlpw3t1h1021gjzhwl226j5k.jpg", 5000, false)
	mediaRepo.Save(context.Background(), avatar1)
	mediaRepo.Save(context.Background(), avatar2)
	mediaRepo.Save(context.Background(), avatar3)
	mediaRepo.Save(context.Background(), avatar4)
	mediaRepo.Save(context.Background(), avatar5)

	// create Profiles (users)
	user1, _ := userProfileService.CreateRealUserProfile(context.Background(), "email000@gmail.com", "+799900011222", "hard password hash", "KokInside", "Stas", "Ignatov", true, nil, models.Gender(1), &avatar1)
	user2, _ := userProfileService.CreateRealUserProfile(context.Background(), "email111@gmail.com", "+179990001122", "hard password hash", "NotationFR", "Ashot", "Ivanovich", true, nil, models.Gender(1), &avatar2)
	user3, _ := userProfileService.CreateRealUserProfile(context.Background(), "email222@gmail.com", "+279990001122", "hard password hash", "Dorifuta", "Potap", "Pogodin", true, nil, models.Gender(1), &avatar3)
	user4, _ := userProfileService.CreateRealUserProfile(context.Background(), "email333@gmail.com", "+379990001122", "hard password hash", "Domik", "Nikita", "Grib", true, nil, models.Gender(1), &avatar4)
	user5, _ := userProfileService.CreateRealUserProfile(context.Background(), "email444@gmail.com", "+479990001122", "hard password hash", "Whiteroom", "Max", "Polezhaev", true, nil, models.Gender(1), &avatar5)

	// create medias
	mediaDesctiption1 := "Media description 1"
	mediaDesctiption2 := "Media description 2"
	mediaDesctiption3 := "Media description 3"
	mediaDesctiption4 := "Media description 4"
	mediaDesctiption5 := "Media description 5"
	mediaDesctiption6 := "Media description 6"
	media1 := models.NewMedia("Media name 1", "jpg", &mediaDesctiption1, "image", "https://moika78.ru/wp-content/uploads/2021/10/ia.jpg", 10241, false)
	media2 := models.NewMedia("Media name 2", "jpg", &mediaDesctiption2, "image", "https://img01.rl0.ru/afisha/e1200x800i/daily.afisha.ru/uploads/images/2/7f/27f311fa0cc7d66b57fd8622350356bf.jpg", 10242, false)
	media3 := models.NewMedia("Media name 3", "jpg", &mediaDesctiption3, "image", "https://img-webcalypt.ru/img/thumb/lg/403/202510/THqUQVWPsZO2aayBVD4Ljc82NTI5PqeTwci2hrvOgxL6FIXy3180PT4Vi9G4m2WWXxDJ2PWuCAKKnayGXRGX8YpxLegrfNnJk06UM4L9e8HZnwr6daNNU8VxijQRvPaZ.jpeg.jpg", 10243, false)
	media4 := models.NewMedia("Media name 4", "jpg", &mediaDesctiption4, "image", "https://icdn.lenta.ru/images/2025/12/09/16/20251209160255692/preview_c8ae7a0a40b6255df50771f8b845c008.jpg", 10244, false)
	media5 := models.NewMedia("Media name 5", "jpg", &mediaDesctiption5, "image", "https://media.tenor.com/nv7JN8Xbx6AAAAAe/%D0%BE%D0%B1%D0%B5%D0%B7%D1%8C%D1%8F%D0%BD%D0%B0-%D0%BE%D0%B1%D0%B5%D0%B7%D1%8C%D1%8F%D0%BD%D0%B0-%D0%BC%D0%B5%D0%BC.png", 10245, false)
	media6 := models.NewMedia("Media name 6", "jpg", &mediaDesctiption6, "image", "https://i.pinimg.com/736x/ba/5e/12/ba5e12e316ac4df8552e637b70677b81.jpg", 10246, false)
	mediaRepo.Save(context.Background(), media1)
	mediaRepo.Save(context.Background(), media2)
	mediaRepo.Save(context.Background(), media3)
	mediaRepo.Save(context.Background(), media4)
	mediaRepo.Save(context.Background(), media5)
	mediaRepo.Save(context.Background(), media6)

	// create posts
	postText1 := "Сегодня я увидел белку, которая, кажется, изучает экономику. Она сидела на ветке, грызла орех и периодически записывала что-то в маленький блокнот. Я пытался понять, о чём она думает, но у меня только появился вопрос: если белка инвестирует в орехи, это считается стабильной валютой или биржей леса? Между тем кот из соседнего двора наблюдал за мной, как будто я был частью научного эксперимента. Вдруг я понял, что сам стал статистикой в беличьих исследованиях. Возможно, завтра я начну носить очки и шляпу, чтобы казаться умнее. Мир странный, а белки — ещё страннее."
	postText2 := "Сегодня на завтрак я решил приготовить омлет с сюрпризом. Выложил яйца на сковороду, а потом вспомнил, что забыл купить молоко. Решил заменить его водой из-под крана. Омлет получился очень живой, почти как маленький аквариум на сковороде. Я смотрел на него и думал: «Если бы он мог говорить, что бы он сказал?» Возможно, что-то вроде: «Спасибо за эту плавучую экскурсию». Кофе я пил с осторожностью, чтобы случайно не добавить в него немного фантазии. День прошёл странно, но вкусно, а омлет остался в памяти как самый мокрый и философский завтрак в истории."
	postText3 := "Сегодня я пытался разговаривать с кактусом. Он, кажется, игнорировал меня, хотя я точно слышал, как он шипел что-то вроде: «Не мешай моему росту». Я подумал, что это хороший урок: иногда молчание говорит больше, чем тысячи слов. Потом я попытался рассказать ему шутку про кактусы, но, видимо, она была слишком колючей. Рядом пёс пытался объяснить мне, что всё это смешно, хотя на самом деле он просто хотел съесть мои носки. Жизнь — это странная смесь растений, животных и моих странных идей, которые, кажется, иногда понимают даже кактусы."
	postText4 := "Вчера я нашёл носок, который, похоже, ушёл в отпуск без меня. Он лежал на полке, выглядел загорелым и немного усталым. Я пытался его вернуть, но он ускользнул между книгами и исчез, как настоящий турист. Понимаю, что носки тоже мечтают об отдыхе, и, возможно, они собираются на маленькие курорты в шкафах. Пёс рядом наблюдал за этим и пытался понять, зачем люди переживают за носки. А я понял, что иногда потеря — это просто способ носкам обрести свободу. Завтра, наверное, куплю новый носок, но буду помнить о старом как о легенде."
	postText5 := "Сегодня я решил попробовать новый способ борьбы со скукой: разговаривать с пылесосом. Он сначала молчал, а потом неожиданно загудел так, будто пытался рассказать секреты вселенной. Я пытался повторять его гудение, но звучало скорее как странная симфония из кухонной утвари. Кофе при этом пролился на стол, создавая импровизированный арт-объект. Кот наблюдал за процессом с видом истинного критика, явно недовольного моими экспериментами. В конце концов я понял, что скука побеждена, хотя пылесос остался загадкой. Иногда странные идеи работают лучше любых планов, особенно если у вас есть кот и немного кофе."
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

	// create reposts
	repost1 := models.NewRepost(user2.ID, uuid.Nil, post1.ID)
	repost2 := models.NewRepost(user3.ID, uuid.Nil, post1.ID)
	repost3 := models.NewRepost(user4.ID, uuid.Nil, post1.ID)
	repost4 := models.NewRepost(user5.ID, uuid.Nil, post2.ID)
	repost5 := models.NewRepost(user3.ID, uuid.Nil, post2.ID)
	repost6 := models.NewRepost(user1.ID, uuid.Nil, post3.ID)
	repost7 := models.NewRepost(user1.ID, uuid.Nil, post4.ID)
	repost8 := models.NewRepost(user1.ID, uuid.Nil, post5.ID)
	repost9 := models.NewRepost(user1.ID, uuid.Nil, post3.ID)

	repostRepo.Save(context.Background(), repost1)
	repostRepo.Save(context.Background(), repost2)
	repostRepo.Save(context.Background(), repost3)
	repostRepo.Save(context.Background(), repost4)
	repostRepo.Save(context.Background(), repost5)
	repostRepo.Save(context.Background(), repost6)
	repostRepo.Save(context.Background(), repost7)
	repostRepo.Save(context.Background(), repost8)
	repostRepo.Save(context.Background(), repost9)
}
