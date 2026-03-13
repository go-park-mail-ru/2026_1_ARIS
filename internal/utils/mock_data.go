package utils

import (
	"context"
	"time"

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
	userAvatar6 := "user avatar 6 description"
	userAvatar7 := "user avatar 7 description"
	userAvatar8 := "user avatar 8 description"
	avatar1 := models.NewMedia("avatar_1_name", "jpg", &userAvatar1, "image", "https://forum.stitch.su/uploads/monthly_2017_10/A.png.b16d1fa2bd3bb388f2122a0c87fbcf5f.png", 5000, false)
	avatar2 := models.NewMedia("avatar_2_name", "jpg", &userAvatar2, "image", "https://i.ibb.co/C3c6HCjb/pop-User1.png", 1000, false)
	avatar3 := models.NewMedia("avatar_3_name", "jpg", &userAvatar3, "image", "https://i.ibb.co/mQvfkNY/pop-User2.png", 2000, false)
	avatar4 := models.NewMedia("avatar_4_name", "jpg", &userAvatar4, "image", "https://i.ibb.co/6RS96KC7/pop-User3.png", 3000, false)
	avatar5 := models.NewMedia("avatar_5_name", "jpg", &userAvatar5, "image", "https://i.ibb.co/mCpKjmxK/pop-User4.png", 4000, false)
	avatar6 := models.NewMedia("avatar_6_name", "jpg", &userAvatar6, "image", "https://i.ibb.co/60HMXYh6/6.jpg", 4000, false)
	avatar7 := models.NewMedia("avatar_7_name", "jpg", &userAvatar7, "image", "https://i.ibb.co/s9rN3qD9/7.jpg", 4000, false)
	avatar8 := models.NewMedia("avatar_8_name", "jpg", &userAvatar8, "image", "https://sun9-5.userapi.com/s/v1/ig2/uGYEtsdSK4QHpAyiRnb5vCasxGZy7dR-MYECGzReWIivHlfmnfQP2DaVY6_UOJHzPG4yzjnVbty6aWqM8kjydEAS.jpg?quality=95&as=32x32,48x48,72x72,108x108,160x160,240x240,360x360,480x480,540x540,640x640&from=bu&cs=640x0", 4000, false)
	//avatar9 := models.NewMedia("avatar_9_name", "jpg", &userAvatar9, "image", "https://i.ibb.co/s9rN3qD9/7.jpg", 4000, false)

	mediaRepo.Save(context.Background(), avatar1)
	mediaRepo.Save(context.Background(), avatar2)
	mediaRepo.Save(context.Background(), avatar3)
	mediaRepo.Save(context.Background(), avatar4)
	mediaRepo.Save(context.Background(), avatar5)
	mediaRepo.Save(context.Background(), avatar6)
	mediaRepo.Save(context.Background(), avatar7)
	mediaRepo.Save(context.Background(), avatar8)

	// create Profiles (users)

	user1, _ := userProfileService.CreateRealUserProfile(context.Background(), "email444@gmail.com", "+479990001122", "hard password hash", "KomandaARIS", "Команда", "АРИС", true, nil, models.Gender(1), &avatar1)
	user2, _ := userProfileService.CreateRealUserProfile(context.Background(), "email000@gmail.com", "+799900011222", "hard password hash", "SergeyShulginenko", "Сергей", "Шульгиненко", true, nil, models.Gender(0), &avatar2)
	user3, _ := userProfileService.CreateRealUserProfile(context.Background(), "email111@gmail.com", "+179990001122", "hard password hash", "AnnaOparina", "Анна", "Опарина", true, nil, models.Gender(1), &avatar3)
	user4, _ := userProfileService.CreateRealUserProfile(context.Background(), "email222@gmail.com", "+279990001122", "hard password hash", "IvanKhvostov", "Иван", "Хвостов", true, nil, models.Gender(0), &avatar4)
	user5, _ := userProfileService.CreateRealUserProfile(context.Background(), "email333@gmail.com", "+379990001122", "hard password hash", "RinatBaikov", "Ринат", "Байков", true, nil, models.Gender(0), &avatar5)
	user6, _ := userProfileService.CreateRealUserProfile(context.Background(), "email444@gmail.com", "+479990001122", "hard password hash", "SofiaSitnichenko", "Софья", "Ситниченко", true, nil, models.Gender(1), &avatar6)
	user7, _ := userProfileService.CreateRealUserProfile(context.Background(), "email444@gmail.com", "+479990001122", "hard password hash", "KonstantinGalanin", "Константин", "Галанин", true, nil, models.Gender(0), &avatar7)
	user8, _ := userProfileService.CreateRealUserProfile(context.Background(), "email444@gmail.com", "+479990001122", "hard password hash", "DaniilKhasyanov", "Даниил", "Хасьянов", true, nil, models.Gender(0), &avatar8)
	//user9, _ := userProfileService.CreateRealUserProfile(context.Background(), "email444@gmail.com", "+479990001122", "hard password hash", "VladislavAlyokhin", "Владислав", "Алехин", true, nil, models.Gender(1), &avatar9)

	// create medias
	mediaDesctiption1 := "Media description 1"
	mediaDesctiption2 := "Media description 2"
	mediaDesctiption3 := "Media description 3"
	//mediaDesctiption4 := "Media description 4"
	mediaDesctiption5 := "Media description 5"
	mediaDesctiption6 := "Media description 6"
	mediaDesctiption7 := "Media description 7"
	mediaDesctiption8 := "Media description 8"
	mediaDesctiption9 := "Media description 9"
	mediaDesctiption10 := "Media description 10"
	mediaDesctiption11 := "Media description 11"
	mediaDesctiption12 := "Media description 12"
	mediaDesctiption13 := "Media description 13"
	mediaDesctiption14 := "Media description 14"
	mediaDesctiption15 := "Media description 15"
	//mediaDesctiption16 := "Media description 16"
	mediaDesctiption17 := "Media description 17"
	mediaDesctiption18 := "Media description 18"
	mediaDesctiption19 := "Media description 19"
	mediaDesctiption20 := "Media description 20"
	mediaDesctiption21 := "Media description 21"
	mediaDesctiption22 := "Media description 22"
	mediaDesctiption23 := "Media description 23"

	media1 := models.NewMedia("Media name 1", "jpg", &mediaDesctiption1, "image", "https://img.freepik.com/free-photo/mountains-lake_1398-1150.jpg", 10246, false)
	media2 := models.NewMedia("Media name 2", "jpg", &mediaDesctiption2, "image", "https://img51994.kanal-o.ru/img/2024-09-09/fmt_81_24_shutterstock_2141488197.jpg", 10246, false)

	media3 := models.NewMedia("Media name 3", "jpg", &mediaDesctiption3, "image", "https://moya-planeta.ru/upload/images/l/eb/e2/ebe21cb5a55a808b104f3d51c3ff96284bae5182.jpg", 10241, false)
	//media4 := models.NewMedia("Media name 4", "jpg", &mediaDesctiption4, "image", "https://www.svitstyle.com.ua/wp-content/uploads/2025/09/pryroda-svitu.jpg", 10242, false)
	media5 := models.NewMedia("Media name 5", "jpg", &mediaDesctiption5, "image", "https://oboitd.ru/images/goods/big/20200125110231_Priroda_10-344.jpg", 10243, false)
	media6 := models.NewMedia("Media name 6", "jpg", &mediaDesctiption6, "image", "https://www.advantour.com/img/kazakhstan/images/nature.jpg", 10244, false)
	media7 := models.NewMedia("Media name 7", "jpg", &mediaDesctiption7, "image", "https://img.goodfon.com/wallpaper/big/5/18/italiia-gory-ozero-peizazh-otrazhenie-priroda.webp", 10245, false)

	media8 := models.NewMedia("Media name 8", "jpg", &mediaDesctiption8, "image", "https://marathonec.ru/wp-content/uploads/2019/07/utrennyaya-probezhka-1.jpg", 10246, false)
	media9 := models.NewMedia("Media name 9", "png", &mediaDesctiption9, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQlVOAM_1swHsumck2XbdMEeEKauDRDiXn86g&s", 10246, false)

	media10 := models.NewMedia("Media name 10", "jpg", &mediaDesctiption10, "image", "https://media.licdn.com/dms/image/v2/D5612AQGuHFW9idrbfw/article-cover_image-shrink_720_1280/article-cover_image-shrink_720_1280/0/1714747679466?e=2147483647&v=beta&t=c9gny1mV4A13_niAAW-2wjP9iglUYtsdoXiMzxfoAxo", 10246, false)

	media11 := models.NewMedia("Media name 11", "png", &mediaDesctiption11, "image", "https://ubifi.net/wp-content/uploads/2025/06/Kinds-of-Internet-Connection.webp", 10246, false)
	media12 := models.NewMedia("Media name 12", "jpg", &mediaDesctiption12, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTs3HaWprjBE3nMGKyH-Myd8D3jK0U0EUqTLw&s", 10246, false)
	media13 := models.NewMedia("Media name 13", "png", &mediaDesctiption13, "image", "https://image.geo.de/30140508/t/r4/v4/w1440/r0/-/internetz-f-209777524-jpg--79960-.jpg", 10246, false)
	media14 := models.NewMedia("Media name 14", "jpg", &mediaDesctiption14, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcS7N25ADdSYwSC-m7qUSqlwPsKd4ALT9F425Q&s", 10246, false)
	media15 := models.NewMedia("Media name 15", "png", &mediaDesctiption15, "image", "https://www.wiwi.uni-wuerzburg.de/fileadmin/_processed_/3/9/csm_computer-1209641_1920_3a999762b2.jpg", 10246, false)
	//media16 := models.NewMedia("Media name 16", "jpg", &mediaDesctiption16, "image", "https://res.cloudinary.com/jerrick/image/upload/v1682443907/64480e82daabca001da8fbbc.jpg", 10246, false)

	media17 := models.NewMedia("Media name 17", "png", &mediaDesctiption17, "image", "https://fitaliancook.com/wp-content/uploads/2025/07/pasta-e-fagioli-rezept-beitragsbild.jpg", 10246, false)
	media18 := models.NewMedia("Media name 18", "jpg", &mediaDesctiption18, "image", "https://eat.de/wp-content/uploads/2025/03/tuerkische-pasta-7014.jpg", 10246, false)
	media19 := models.NewMedia("Media name 19", "png", &mediaDesctiption19, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcT0kWh_aX8DW5H8BkMJ3xqzXsRXPY2kyZu5ww&s", 10246, false)
	media20 := models.NewMedia("Media name 20", "jpg", &mediaDesctiption20, "image", "https://images.gastronom.ru/TYj7-7529vyMsVom2kYJQl8MFrkWsrOY5hgaQPa1zsk/pr:article-cover-image/g:ce/rs:auto:0:0:0/L2Ntcy9hbGwtaW1hZ2VzL2IzY2RlN2ZjLTgzZjEtNGJlYi1iOGZmLWZhMzM3YzY1ODFlYy5qcGc.webp", 10246, false)

	media21 := models.NewMedia("Media name 21", "png", &mediaDesctiption21, "image", "https://boxru.ru/upload/resize_cache/iblock/1af/400_400_140cd750bba9870f18aada2478b24840a/q3bxff3vhe8iljlcbpn4jlbx3szt2p1w.jpg", 10246, false)
	media22 := models.NewMedia("Media name 22", "jpg", &mediaDesctiption22, "image", "https://s1.stc.all.kpcdn.net/putevoditel/projectid_346574/images/tild3037-3837-4461-a261-663863336336__photo.jpg", 10246, false)

	media23 := models.NewMedia("Media name 23", "png", &mediaDesctiption23, "image", "https://space-pm.ru/uploads/market/stati/small/1719495934.jpg", 10246, false)

	mediaRepo.Save(context.Background(), media1)
	mediaRepo.Save(context.Background(), media2)
	mediaRepo.Save(context.Background(), media3)
	//mediaRepo.Save(context.Background(), media4)
	mediaRepo.Save(context.Background(), media5)
	mediaRepo.Save(context.Background(), media6)
	mediaRepo.Save(context.Background(), media7)
	mediaRepo.Save(context.Background(), media8)
	mediaRepo.Save(context.Background(), media9)
	mediaRepo.Save(context.Background(), media10)
	mediaRepo.Save(context.Background(), media11)
	mediaRepo.Save(context.Background(), media12)
	mediaRepo.Save(context.Background(), media13)
	mediaRepo.Save(context.Background(), media14)
	mediaRepo.Save(context.Background(), media15)
	//mediaRepo.Save(context.Background(), media16)
	mediaRepo.Save(context.Background(), media17)
	mediaRepo.Save(context.Background(), media18)
	mediaRepo.Save(context.Background(), media19)
	mediaRepo.Save(context.Background(), media20)
	mediaRepo.Save(context.Background(), media21)
	mediaRepo.Save(context.Background(), media22)
	mediaRepo.Save(context.Background(), media23)

	// create posts

	postText1 := `Привет! Добро пожаловать в ARIS :) Мы хотели создать нашу социальную сеть в том виде, как она задумывалась изначально - с акцентом на общение со знакомыми нам людьми и поиском новых, схожих с нами по интересам.

После регистрации у тебя появится своя персональная страничка и лента.

К этому сообщению мы прикрепили картинки. После регистрации ты сможешь публиковать такие же посты со своими изображениями.`
	postText2 := `Это второй пост в ленте. Лента — это место, куда ты сможешь заглядывать за новыми постами, которые оставили твои друзья или другие пользователей нашей с тобой социальной сети.

Лента может отображаться:

— по времени
— по рекомендациям ("Для вас")

Попробуй попереключать режим в левом меню :) 

В твоем случае посты поменяются местами, но у авторизованных пользователей это две совершенно разные ленты.`
	postText3 := `Сегодня впервые за долгое время решил выйти на пробежку утром, а не вечером.

Город в это время выглядит совсем по-другому: почти нет людей, воздух свежий, а солнце только начинает подниматься.

Пробежал всего пять километров, но ощущение будто день уже начался правильно.

Иногда кажется, что именно такие маленькие привычки сильнее всего меняют жизнь.

Думаю попробовать бегать утром хотя бы пару раз в неделю.

А вы когда предпочитаете тренироваться — утром или вечером?`
	postText4 := `Сегодня весь день пыталась разобраться с новой библиотекой для фронтенда.

Сначала всё казалось довольно простым, но потом начались неожиданные ошибки.

Самое интересное, что проблема оказалась всего в одной строке кода.

Каждый раз удивляюсь, как одна мелочь может сломать половину приложения.

Зато теперь стало гораздо понятнее, как работает архитектура проекта.

Люблю это ощущение, когда после долгих попыток всё наконец начинает работать.`
	postText5 := `Недавно начала читать книгу про историю интернета.

Оказывается, многие вещи, которые сегодня кажутся очевидными, появлялись почти случайно.

Например, первые социальные сети выглядели совсем иначе и были очень простыми.

Никаких алгоритмов рекомендаций, сложных интерфейсов и бесконечных лент.

Просто люди писали сообщения и общались.

Интересно наблюдать, как технологии меняют то, как мы взаимодействуем друг с другом.

Иногда полезно посмотреть на истоки современных сервисов.`
	postText6 := `Сегодня попробовала приготовить новый рецепт пасты.

На удивление получилось намного лучше, чем ожидала.

Иногда кажется, что готовка — это почти как программирование.

Есть набор ингредиентов, есть последовательность действий и всегда есть шанс что-то испортить.

Но когда всё получается — результат радует гораздо больше.

Теперь думаю попробовать ещё пару похожих рецептов.

Если у вас есть любимые блюда, которые легко приготовить — поделитесь.
`
	postText7 := `Последние пару недель пытаюсь меньше сидеть в телефоне.

Заметил, что если просто убрать уведомления, то времени становится гораздо больше.

Начал читать книги по вечерам вместо того, чтобы бесконечно листать ленты.

Сначала было непривычно, но теперь даже нравится.

Появилось ощущение, что день стал длиннее.

Иногда полезно немного замедлиться и отвлечься от экранов.

А у вас получается ограничивать время в соцсетях?`
	postText8 := `Сегодня решил немного изменить рабочую обстановку и поработать не дома, а в кофейне.

Иногда смена места помогает взглянуть на задачи по-новому. Вокруг шум, люди разговаривают, играет музыка — но при этом почему-то легче сосредоточиться.

Удалось закрыть несколько задач, до которых долго не доходили руки.

Наверное, буду иногда устраивать такие небольшие “рабочие вылазки”.

А вы где предпочитаете работать или учиться — дома, в офисе или в каких-нибудь спокойных местах вроде кофеен?`

	post1 := models.NewPost(&postText1, *user1, true, true)
	post2 := models.NewPost(&postText2, *user1, true, true)
	post3 := models.NewPost(&postText3, *user3, true, false)
	post4 := models.NewPost(&postText4, *user4, true, false)
	post5 := models.NewPost(&postText5, *user5, true, false)
	post6 := models.NewPost(&postText6, *user6, true, false)
	post7 := models.NewPost(&postText7, *user7, true, false)
	post8 := models.NewPost(&postText8, *user8, true, false)
	now := time.Now()

	post1.CreatedAt = now.Add(0 * time.Minute)
	post1.UpdatedAt = post1.CreatedAt

	post2.CreatedAt = now.Add(-1 * time.Minute)
	post2.UpdatedAt = post2.CreatedAt

	post3.CreatedAt = now.Add(-1 * time.Hour)
	post3.UpdatedAt = post3.CreatedAt

	post4.CreatedAt = now.Add(-2 * time.Hour)
	post4.UpdatedAt = post4.CreatedAt

	post5.CreatedAt = now.Add(-5 * time.Hour)
	post5.UpdatedAt = post5.CreatedAt

	post6.CreatedAt = now.Add(-10 * time.Hour)
	post6.UpdatedAt = post6.CreatedAt

	post7.CreatedAt = now.Add(-24 * time.Hour)
	post7.UpdatedAt = post7.CreatedAt

	post8.CreatedAt = now.Add(-48 * time.Hour)
	post8.UpdatedAt = post8.CreatedAt
	postService.Save(context.Background(), post1)
	postService.Save(context.Background(), post2)
	postService.Save(context.Background(), post3)
	postService.Save(context.Background(), post4)
	postService.Save(context.Background(), post5)
	postService.Save(context.Background(), post6)
	postService.Save(context.Background(), post7)
	postService.Save(context.Background(), post8)

	// connect post with medias to get PostWithMedia
	postWithMediaRepo.Save(post1, media1, 0)
	postWithMediaRepo.Save(post1, media2, 1)

	postWithMediaRepo.Save(post2, media3, 0)
	//postWithMediaRepo.Save(post2, media4, 1)
	postWithMediaRepo.Save(post2, media5, 2)
	postWithMediaRepo.Save(post2, media6, 3)
	postWithMediaRepo.Save(post2, media7, 4)

	postWithMediaRepo.Save(post3, media8, 0)
	postWithMediaRepo.Save(post3, media9, 1)

	postWithMediaRepo.Save(post4, media10, 0)

	postWithMediaRepo.Save(post5, media11, 0)
	postWithMediaRepo.Save(post5, media12, 1)
	postWithMediaRepo.Save(post5, media13, 2)
	postWithMediaRepo.Save(post5, media14, 3)
	postWithMediaRepo.Save(post5, media15, 4)
	//postWithMediaRepo.Save(post5, media16, 5)

	postWithMediaRepo.Save(post6, media17, 0)
	postWithMediaRepo.Save(post6, media18, 1)
	postWithMediaRepo.Save(post6, media19, 2)
	postWithMediaRepo.Save(post6, media20, 3)

	postWithMediaRepo.Save(post7, media21, 0)
	postWithMediaRepo.Save(post7, media22, 1)

	postWithMediaRepo.Save(post8, media23, 1)

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
