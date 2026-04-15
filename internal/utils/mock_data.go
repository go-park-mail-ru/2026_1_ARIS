package utils

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models" // добавлен импорт
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	postrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
	postservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func MakeMock(mediaRepo media.MediaRepo,
	userProfileService user.UserService,
	postService postservice.PostService,
	postWithMediaRepo postrepo.PostWithMediaRepo,
	commentRepo comment.CommentRepo,
	repostRepo repost.RepostRepo,
	chatRepo *chat.SQLChatRepo, // изменён тип
	logger *zap.Logger,
) {

	// create Profiles (users)

	email1 := "email4441@gmail.com"
	email2 := "email0002@gmail.com"
	email3 := "email1113@gmail.com"
	email4 := "email2224@gmail.com"
	email5 := "email3335@gmail.com"
	email6 := "email4446@gmail.com"
	email7 := "email4447@gmail.com"
	email8 := "email4448@gmail.com"

	phone1 := "+479990101122"
	phone2 := "+799900211222"
	phone3 := "+179990301122"
	phone4 := "+279990401122"
	phone5 := "+379990501122"
	phone6 := "+479990601122"
	phone7 := "+479990701122"
	phone8 := "+479990901122"

	birthdayDate1, _ := time.Parse("02/01/2006", "24/02/2005")
	birthdayDate2, _ := time.Parse("02/01/2006", "24/02/2005")
	birthdayDate3, _ := time.Parse("02/01/2006", "24/02/2005")
	birthdayDate4, _ := time.Parse("02/01/2006", "24/02/2005")
	birthdayDate5, _ := time.Parse("02/01/2006", "24/02/2005")
	birthdayDate6, _ := time.Parse("02/01/2006", "24/02/2005")
	birthdayDate7, _ := time.Parse("02/01/2006", "24/02/2005")
	birthdayDate8, _ := time.Parse("02/01/2006", "24/02/2005")

	// пользователь без аватарки, чтобы создавать все другие аватарки
	user1, err := userProfileService.CreateRealUserProfile(context.Background(), &email1, &phone1, "hard password hash", "KomandaARIS", "Команда", "АРИС", birthdayDate1, models.Gender("male"), nil)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}

	// create user avatars
	// userAvatar1 := "user avatar 1 description"
	userAvatar2 := "user avatar 2 description"
	userAvatar3 := "user avatar 3 description"
	userAvatar4 := "user avatar 4 description"
	userAvatar5 := "user avatar 5 description"
	userAvatar6 := "user avatar 6 description"
	userAvatar7 := "user avatar 7 description"
	userAvatar8 := "user avatar 8 description"

	//avatar1 := models.NewMedia("avatar_1_name", "jpg", uuid.New(), &userAvatar1, "image", "/image-proxy?url=https://forum.stitch.su/uploads/monthly_2017_10/A.png.b16d1fa2bd3bb388f2122a0c87fbcf5f.png", 1)
	avatar2 := models.NewMedia("avatar_2_name", "jpg", uuid.New(), &userAvatar2, "image", "/image-proxy?url=https://i.ibb.co/C3c6HCjb/pop-User1.png", 1)
	avatar3 := models.NewMedia("avatar_3_name", "jpg", uuid.New(), &userAvatar3, "image", "/image-proxy?url=https://i.ibb.co/mQvfkNY/pop-User2.png", 1)
	avatar4 := models.NewMedia("avatar_4_name", "jpg", uuid.New(), &userAvatar4, "image", "/image-proxy?url=https://i.ibb.co/6RS96KC7/pop-User3.png", 1)
	avatar5 := models.NewMedia("avatar_5_name", "jpg", uuid.New(), &userAvatar5, "image", "/image-proxy?url=https://i.ibb.co/mCpKjmxK/pop-User4.png", 1)
	avatar6 := models.NewMedia("avatar_6_name", "jpg", uuid.New(), &userAvatar6, "image", "/image-proxy?url=https://i.ibb.co/60HMXYh6/6.jpg", 1)
	avatar7 := models.NewMedia("avatar_7_name", "jpg", uuid.New(), &userAvatar7, "image", "/image-proxy?url=https://i.ibb.co/s9rN3qD9/7.jpg", 1)
	avatar8 := models.NewMedia("avatar_8_name", "jpg", uuid.New(), &userAvatar8, "image", "/image-proxy?url=https://sun9-5.userapi.com/s/v1/ig2/uGYEtsdSK4QHpAyiRnb5vCasxGZy7dR-MYECGzReWIivHlfmnfQP2DaVY6_UOJHzPG4yzjnVbty6aWqM8kjydEAS.jpg?quality=95&as=32x32,48x48,72x72,108x108,160x160,240x240,360x360,480x480,540x540,640x640&from=bu&cs=640x0", 1)

	//avatar9 := models.NewMedia("avatar_9_name", "jpg", &userAvatar9, "image", "https://i.ibb.co/s9rN3qD9/7.jpg", 4000, false)

	// avatar1ID, err := mediaRepo.Save(context.Background(), *avatar1)
	// if err != nil {
	// 	logger.Info("faild saving", zap.Error(err))
	// 	return
	// }

	avatar2ID, err := mediaRepo.Save(context.Background(), *avatar2)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	avatar3ID, err := mediaRepo.Save(context.Background(), *avatar3)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	avatar4ID, err := mediaRepo.Save(context.Background(), *avatar4)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	avatar5ID, err := mediaRepo.Save(context.Background(), *avatar5)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	avatar6ID, err := mediaRepo.Save(context.Background(), *avatar6)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	avatar7ID, err := mediaRepo.Save(context.Background(), *avatar7)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	avatar8ID, err := mediaRepo.Save(context.Background(), *avatar8)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	logger.Info("success avatars creation")

	user2, err := userProfileService.CreateRealUserProfile(context.Background(), &email2, &phone2, "hard password hash", "SergeyShulginenko", "Сергей", "Шульгиненко", birthdayDate2, models.Gender("female"), &avatar2ID)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}

	user3, err := userProfileService.CreateRealUserProfile(context.Background(), &email3, &phone3, "hard password hash", "AnnaOparina", "Анна", "Опарина", birthdayDate3, models.Gender("male"), &avatar3ID)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}

	user4, err := userProfileService.CreateRealUserProfile(context.Background(), &email4, &phone4, "hard password hash", "IvanKhvostov", "Иван", "Хвостов", birthdayDate4, models.Gender("female"), &avatar4ID)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}
	user5, err := userProfileService.CreateRealUserProfile(context.Background(), &email5, &phone5, "hard password hash", "RinatBaikov", "Ринат", "Байков", birthdayDate5, models.Gender("male"), &avatar5ID)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}

	user6, err := userProfileService.CreateRealUserProfile(context.Background(), &email6, &phone6, "hard password hash", "SofiaSitnichenko", "Софья", "Ситниченко", birthdayDate6, models.Gender("female"), &avatar6ID)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}

	user7, err := userProfileService.CreateRealUserProfile(context.Background(), &email7, &phone7, "hard password hash", "KonstantinGalanin", "Константин", "Галанин", birthdayDate7, models.Gender("male"), &avatar7ID)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}

	user8, err := userProfileService.CreateRealUserProfile(context.Background(), &email8, &phone8, "hard password hash", "DaniilKhasyanov", "Даниил", "Хасьянов", birthdayDate8, models.Gender("female"), &avatar8ID)
	if err != nil {
		logger.Info("faild to save user", zap.Error(err))
		return
	}

	logger.Info("success users creation")

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

	media1 := models.NewMedia("Media name 1", "jpg", uuid.New(), &mediaDesctiption1, "image", "https://img.freepik.com/free-photo/mountains-lake_1398-1150.jpg", 1)
	media2 := models.NewMedia("Media name 2", "jpg", uuid.New(), &mediaDesctiption2, "image", "https://img51994.kanal-o.ru/img/2024-09-09/fmt_81_24_shutterstock_2141488197.jpg", 1)

	media3 := models.NewMedia("Media name 3", "jpg", uuid.New(), &mediaDesctiption3, "image", "https://moya-planeta.ru/upload/images/l/eb/e2/ebe21cb5a55a808b104f3d51c3ff96284bae5182.jpg", 1)
	//media4 := models.NewMedia("Media name 4", "jpg", uuid.New(), &mediaDesctiption4, "image", "https://www.svitstyle.com.ua/wp-content/uploads/2025/09/pryroda-svitu.jpg", 10242, false)
	media5 := models.NewMedia("Media name 5", "jpg", uuid.New(), &mediaDesctiption5, "image", "https://oboitd.ru/images/goods/big/20200125110231_Priroda_10-344.jpg", 1)
	media6 := models.NewMedia("Media name 6", "jpg", uuid.New(), &mediaDesctiption6, "image", "https://www.advantour.com/img/kazakhstan/images/nature.jpg", 1)
	media7 := models.NewMedia("Media name 7", "jpg", uuid.New(), &mediaDesctiption7, "image", "https://img.goodfon.com/wallpaper/big/5/18/italiia-gory-ozero-peizazh-otrazhenie-priroda.webp", 1)

	media8 := models.NewMedia("Media name 8", "jpg", uuid.New(), &mediaDesctiption8, "image", "https://marathonec.ru/wp-content/uploads/2019/07/utrennyaya-probezhka-1.jpg", 1)
	media9 := models.NewMedia("Media name 9", "png", uuid.New(), &mediaDesctiption9, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQlVOAM_1swHsumck2XbdMEeEKauDRDiXn86g&s", 2)

	media10 := models.NewMedia("Media name 10", "jpg", uuid.New(), &mediaDesctiption10, "image", "https://media.licdn.com/dms/image/v2/D5612AQGuHFW9idrbfw/article-cover_image-shrink_720_1280/article-cover_image-shrink_720_1280/0/1714747679466?e=2147483647&v=beta&t=c9gny1mV4A13_niAAW-2wjP9iglUYtsdoXiMzxfoAxo", 2)

	media11 := models.NewMedia("Media name 11", "png", uuid.New(), &mediaDesctiption11, "image", "https://ubifi.net/wp-content/uploads/2025/06/Kinds-of-Internet-Connection.webp", 2)
	media12 := models.NewMedia("Media name 12", "jpg", uuid.New(), &mediaDesctiption12, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTs3HaWprjBE3nMGKyH-Myd8D3jK0U0EUqTLw&s", 2)
	media13 := models.NewMedia("Media name 13", "png", uuid.New(), &mediaDesctiption13, "image", "https://image.geo.de/30140508/t/r4/v4/w1440/r0/-/internetz-f-209777524-jpg--79960-.jpg", 2)
	media14 := models.NewMedia("Media name 14", "jpg", uuid.New(), &mediaDesctiption14, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcS7N25ADdSYwSC-m7qUSqlwPsKd4ALT9F425Q&s", 3)
	media15 := models.NewMedia("Media name 15", "png", uuid.New(), &mediaDesctiption15, "image", "https://www.wiwi.uni-wuerzburg.de/fileadmin/_processed_/3/9/csm_computer-1209641_1920_3a999762b2.jpg", 3)
	//media16 := models.NewMedia("Media name 16", "jpg", uuid.New(), &mediaDesctiption16, "image", "https://res.cloudinary.com/jerrick/image/upload/v1682443907/64480e82daabca001da8fbbc.jpg", 10246, false)

	media17 := models.NewMedia("Media name 17", "png", uuid.New(), &mediaDesctiption17, "image", "https://fitaliancook.com/wp-content/uploads/2025/07/pasta-e-fagioli-rezept-beitragsbild.jpg", 3)
	media18 := models.NewMedia("Media name 18", "jpg", uuid.New(), &mediaDesctiption18, "image", "https://eat.de/wp-content/uploads/2025/03/tuerkische-pasta-7014.jpg", 3)
	media19 := models.NewMedia("Media name 19", "png", uuid.New(), &mediaDesctiption19, "image", "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcT0kWh_aX8DW5H8BkMJ3xqzXsRXPY2kyZu5ww&s", 3)
	media20 := models.NewMedia("Media name 20", "jpg", uuid.New(), &mediaDesctiption20, "image", "https://images.gastronom.ru/TYj7-7529vyMsVom2kYJQl8MFrkWsrOY5hgaQPa1zsk/pr:article-cover-image/g:ce/rs:auto:0:0:0/L2Ntcy9hbGwtaW1hZ2VzL2IzY2RlN2ZjLTgzZjEtNGJlYi1iOGZmLWZhMzM3YzY1ODFlYy5qcGc.webp", 4)

	media21 := models.NewMedia("Media name 21", "png", uuid.New(), &mediaDesctiption21, "image", "https://boxru.ru/upload/resize_cache/iblock/1af/400_400_140cd750bba9870f18aada2478b24840a/q3bxff3vhe8iljlcbpn4jlbx3szt2p1w.jpg", 4)
	media22 := models.NewMedia("Media name 22", "jpg", uuid.New(), &mediaDesctiption22, "image", "https://s1.stc.all.kpcdn.net/putevoditel/projectid_346574/images/tild3037-3837-4461-a261-663863336336__photo.jpg", 4)

	media23 := models.NewMedia("Media name 23", "png", uuid.New(), &mediaDesctiption23, "image", "https://space-pm.ru/uploads/market/stati/small/1719495934.jpg", 4)

	media1ID, err := mediaRepo.Save(context.Background(), *media1)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media2ID, err := mediaRepo.Save(context.Background(), *media2)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media3ID, err := mediaRepo.Save(context.Background(), *media3)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	//mediaRepo.Save(context.Background(), media4)

	media5ID, err := mediaRepo.Save(context.Background(), *media5)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media6ID, err := mediaRepo.Save(context.Background(), *media6)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media7ID, err := mediaRepo.Save(context.Background(), *media7)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media8ID, err := mediaRepo.Save(context.Background(), *media8)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media9ID, err := mediaRepo.Save(context.Background(), *media9)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media10ID, err := mediaRepo.Save(context.Background(), *media10)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media11ID, err := mediaRepo.Save(context.Background(), *media11)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media12ID, err := mediaRepo.Save(context.Background(), *media12)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media13ID, err := mediaRepo.Save(context.Background(), *media13)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media14ID, err := mediaRepo.Save(context.Background(), *media14)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media15ID, err := mediaRepo.Save(context.Background(), *media15)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	//mediaRepo.Save(context.Background(), media16)

	media17ID, err := mediaRepo.Save(context.Background(), *media17)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media18ID, err := mediaRepo.Save(context.Background(), *media18)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media19ID, err := mediaRepo.Save(context.Background(), *media19)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media20ID, err := mediaRepo.Save(context.Background(), *media20)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media21ID, err := mediaRepo.Save(context.Background(), *media21)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media22ID, err := mediaRepo.Save(context.Background(), *media22)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	media23ID, err := mediaRepo.Save(context.Background(), *media23)
	if err != nil {
		logger.Info("faild to save media", zap.Error(err))
		return
	}

	logger.Info("success media creation")

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
	postText3 := `Сегодня впервые за долгое время решила выйти на пробежку утром, а не вечером.

Город в это время выглядит совсем по-другому: почти нет людей, воздух свежий, а солнце только начинает подниматься.

Пробежала всего пять километров, но ощущение будто день уже начался правильно.

Иногда кажется, что именно такие маленькие привычки сильнее всего меняют жизнь.

Думаю попробовать бегать утром хотя бы пару раз в неделю.

А вы когда предпочитаете тренироваться — утром или вечером?`
	postText4 := `Сегодня весь день пытался разобраться с новой библиотекой для фронтенда.

Сначала всё казалось довольно простым, но потом начались неожиданные ошибки.

Самое интересное, что проблема оказалась всего в одной строке кода.

Каждый раз удивляюсь, как одна мелочь может сломать половину приложения.

Зато теперь стало гораздо понятнее, как работает архитектура проекта.

Люблю это ощущение, когда после долгих попыток всё наконец начинает работать.`
	postText5 := `Недавно начал читать книгу про историю интернета.

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

	post1 := models.NewPost(&postText1, user1.ID, true, true)
	post2 := models.NewPost(&postText2, user1.ID, true, true)
	post3 := models.NewPost(&postText3, user3.ID, false, true)
	post4 := models.NewPost(&postText4, user4.ID, false, true)
	post5 := models.NewPost(&postText5, user5.ID, false, true)
	post6 := models.NewPost(&postText6, user6.ID, false, true)
	post7 := models.NewPost(&postText7, user7.ID, false, true)
	post8 := models.NewPost(&postText8, user8.ID, false, true)
	now := time.Now()

	post1.CreatedAt = now.Add(0 * time.Minute)
	post1.UpdatedAt = post1.CreatedAt
	post1.IsPublicDemo = true

	post2.CreatedAt = now.Add(-1 * time.Minute)
	post2.UpdatedAt = post2.CreatedAt
	post2.IsPublicDemo = true

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

	post1ID, err := postService.Save(context.Background(), *post1)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	post2ID, err := postService.Save(context.Background(), *post2)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	post3ID, err := postService.Save(context.Background(), *post3)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	post4ID, err := postService.Save(context.Background(), *post4)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	post5ID, err := postService.Save(context.Background(), *post5)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	post6ID, err := postService.Save(context.Background(), *post6)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	post7ID, err := postService.Save(context.Background(), *post7)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	post8ID, err := postService.Save(context.Background(), *post8)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	logger.Info("success posts creation")

	postWithMedia1 := models.NewPostWithMedia(post1ID, media1ID, 0)
	postWithMedia2 := models.NewPostWithMedia(post1ID, media2ID, 1)

	postWithMedia3 := models.NewPostWithMedia(post2ID, media3ID, 0)
	postWithMedia4 := models.NewPostWithMedia(post2ID, media5ID, 2)
	postWithMedia5 := models.NewPostWithMedia(post2ID, media6ID, 3)
	postWithMedia6 := models.NewPostWithMedia(post2ID, media7ID, 4)

	postWithMedia7 := models.NewPostWithMedia(post3ID, media8ID, 0)
	postWithMedia8 := models.NewPostWithMedia(post3ID, media9ID, 1)

	postWithMedia9 := models.NewPostWithMedia(post4ID, media10ID, 0)

	postWithMedia10 := models.NewPostWithMedia(post5ID, media11ID, 0)
	postWithMedia11 := models.NewPostWithMedia(post5ID, media12ID, 1)
	postWithMedia12 := models.NewPostWithMedia(post5ID, media13ID, 2)
	postWithMedia13 := models.NewPostWithMedia(post5ID, media14ID, 3)
	postWithMedia14 := models.NewPostWithMedia(post5ID, media15ID, 4)

	postWithMedia15 := models.NewPostWithMedia(post6ID, media17ID, 0)
	postWithMedia16 := models.NewPostWithMedia(post6ID, media18ID, 1)
	postWithMedia17 := models.NewPostWithMedia(post6ID, media19ID, 2)
	postWithMedia18 := models.NewPostWithMedia(post6ID, media20ID, 3)

	postWithMedia19 := models.NewPostWithMedia(post7ID, media21ID, 0)
	postWithMedia20 := models.NewPostWithMedia(post7ID, media22ID, 1)

	postWithMedia21 := models.NewPostWithMedia(post8ID, media23ID, 1)

	// connect post with medias to get PostWithMedia
	postWithMediaRepo.Save(context.Background(), *postWithMedia1)
	postWithMediaRepo.Save(context.Background(), *postWithMedia2)

	//postWithMediaRepo.Save(post2, media4, 1)
	err = postWithMediaRepo.Save(context.Background(), *postWithMedia3)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia4)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia5)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia6)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia7)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia8)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia9)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia10)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia11)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia12)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia13)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia14)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia15)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia16)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia17)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia18)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia19)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia20)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	err = postWithMediaRepo.Save(context.Background(), *postWithMedia21)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	logger.Info("success posts-with-media creation")

	//postWithMediaRepo.Save(post5, media16, 5)

	like1 := models.NewLikeToPost(post1.ID, user4.ID)
	like2 := models.NewLikeToPost(post2.ID, user5.ID)
	like3 := models.NewLikeToPost(post3.ID, user1.ID)
	like4 := models.NewLikeToPost(post4.ID, user2.ID)
	like5 := models.NewLikeToPost(post5.ID, user3.ID)
	like6 := models.NewLikeToPost(post6.ID, user3.ID)
	likeRepo := like.NewLikeRepo()

	_, err = likeRepo.Save(context.Background(), *like1)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = likeRepo.Save(context.Background(), *like2)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = likeRepo.Save(context.Background(), *like3)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = likeRepo.Save(context.Background(), *like4)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = likeRepo.Save(context.Background(), *like5)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = likeRepo.Save(context.Background(), *like6)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	logger.Info("success likes creation")

	commentText1 := "comment 1"
	commentText2 := "comment 2"
	commentText3 := "comment 3"
	commentText4 := "comment 4"
	commentText5 := "comment 5"
	commentText6 := "comment 6"

	comment1 := models.NewComment(&commentText1, post1ID, nil, nil, user2.ID)
	comment2 := models.NewComment(&commentText2, post2ID, nil, nil, user3.ID)
	comment3 := models.NewComment(&commentText3, post3ID, nil, nil, user4.ID)
	comment4 := models.NewComment(&commentText4, post4ID, nil, nil, user5.ID)
	comment5 := models.NewComment(&commentText5, post5ID, nil, nil, user1.ID)
	comment6 := models.NewComment(&commentText6, post1ID, nil, nil, user4.ID)

	_, err = commentRepo.Save(context.Background(), *comment1)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = commentRepo.Save(context.Background(), *comment2)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = commentRepo.Save(context.Background(), *comment3)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = commentRepo.Save(context.Background(), *comment4)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = commentRepo.Save(context.Background(), *comment5)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	_, err = commentRepo.Save(context.Background(), *comment6)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	logger.Info("success comments creation")

	chat1 := models.NewChat(models.ChatType("personal"), "chat 1 title", nil)
	chat2 := models.NewChat(models.ChatType("personal"), "chat 2 title", nil)
	chat3 := models.NewChat(models.ChatType("personal"), "chat 3 title", nil)
	chat4 := models.NewChat(models.ChatType("personal"), "chat 4 title", nil)
	chat5 := models.NewChat(models.ChatType("personal"), "chat 5 title", nil)

	if err := chatRepo.Save(context.Background(), chat1); err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	if err := chatRepo.Save(context.Background(), chat2); err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	if err := chatRepo.Save(context.Background(), chat3); err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	if err := chatRepo.Save(context.Background(), chat4); err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	if err := chatRepo.Save(context.Background(), chat5); err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	logger.Info("success chats creation")

	// create reposts
	repost1 := models.NewRepost(user2.ID, chat1.ID, post1ID)
	repost2 := models.NewRepost(user3.ID, chat2.ID, post1ID)
	repost3 := models.NewRepost(user4.ID, chat3.ID, post1ID)
	repost4 := models.NewRepost(user5.ID, chat4.ID, post2ID)
	repost5 := models.NewRepost(user3.ID, chat5.ID, post2ID)
	repost6 := models.NewRepost(user1.ID, chat1.ID, post3ID)
	repost7 := models.NewRepost(user1.ID, chat2.ID, post4ID)
	repost8 := models.NewRepost(user1.ID, chat3.ID, post5ID)
	repost9 := models.NewRepost(user1.ID, chat4.ID, post3ID)

	_, err = repostRepo.Save(context.Background(), *repost1)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost2)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost3)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost4)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost5)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost6)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost7)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost8)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}
	_, err = repostRepo.Save(context.Background(), *repost9)
	if err != nil {
		logger.Info("faild saving", zap.Error(err))
		return
	}

	logger.Info("success reposts creation")

}
