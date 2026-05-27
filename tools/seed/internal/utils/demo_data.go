package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const demoPassword = "password"

type demoUser struct {
	Username    string
	Email       string
	Phone       string
	FirstName   string
	LastName    string
	Gender      string
	Birthday    string
	Bio         string
	Town        string
	NativeTown  string
	Institution string
	StudyGroup  string
	Company     string
	JobTitle    string
	Interests   string
	FavMusic    string
	AvatarURL   string
}

type demoUserRecord struct {
	AccountID int64
	ProfileID int64
}

// MakeDemoData creates a repeatable dataset for manual review and demos.
func MakeDemoData(ctx context.Context, db *pgxpool.Pool) error {
	users, err := seedDemoUsers(ctx, db)
	if err != nil {
		return err
	}

	media, err := seedDemoMedia(ctx, db, users["demoowner"].ProfileID)
	if err != nil {
		return err
	}

	if err := seedDemoFriendships(ctx, db, users); err != nil {
		return err
	}

	if err := seedDemoChats(ctx, db, users); err != nil {
		return err
	}

	if err := seedDemoUserPosts(ctx, db, users, media); err != nil {
		return err
	}

	if err := seedDemoCommunities(ctx, db, users, media); err != nil {
		return err
	}

	return nil
}

func seedDemoUsers(ctx context.Context, db *pgxpool.Pool) (map[string]demoUserRecord, error) {
	people := []demoUser{
		{
			Username:    "demoowner",
			Email:       "demo.owner@aris.test",
			Phone:       "+79001000001",
			FirstName:   "Олег",
			LastName:    "Владелец",
			Gender:      "male",
			Birthday:    "2000-01-10",
			Bio:         "Проверяет роли владельца, публикации и управление сообществами.",
			Town:        "Москва",
			Institution: "МГТУ им. Н. Э. Баумана",
			AvatarURL:   "https://i.ibb.co/C3c6HCjb/pop-User1.png",
		},
		{
			Username:    "demoadmin",
			Email:       "demo.admin@aris.test",
			Phone:       "+79001000002",
			FirstName:   "Алина",
			LastName:    "Админова",
			Gender:      "female",
			Birthday:    "2001-02-11",
			Bio:         "Администратор демо-сообщества.",
			Town:        "Санкт-Петербург",
			Institution: "ИТМО",
			AvatarURL:   "https://i.ibb.co/mQvfkNY/pop-User2.png",
		},
		{
			Username:    "demomoderator",
			Email:       "demo.moderator@aris.test",
			Phone:       "+79001000003",
			FirstName:   "Марина",
			LastName:    "Модераторова",
			Gender:      "female",
			Birthday:    "2002-03-12",
			Bio:         "Модератор, которому можно менять права и проверять редактирование постов.",
			Town:        "Казань",
			Institution: "КФУ",
			AvatarURL:   "https://i.ibb.co/6RS96KC7/pop-User3.png",
		},
		{
			Username:    "demomember",
			Email:       "demo.member@aris.test",
			Phone:       "+79001000004",
			FirstName:   "Иван",
			LastName:    "Участников",
			Gender:      "male",
			Birthday:    "2003-04-13",
			Bio:         "Обычный участник для проверки заявок, друзей и чатов.",
			Town:        "Екатеринбург",
			Institution: "УрФУ",
			AvatarURL:   "https://i.ibb.co/mCpKjmxK/pop-User4.png",
		},
		{
			Username:    "demoblocked",
			Email:       "demo.blocked@aris.test",
			Phone:       "+79001000005",
			FirstName:   "Борис",
			LastName:    "Заблокирован",
			Gender:      "male",
			Birthday:    "2004-05-14",
			Bio:         "Пользователь для проверки блокировки в сообществе.",
			Town:        "Новосибирск",
			Institution: "НГУ",
			AvatarURL:   "https://i.ibb.co/60HMXYh6/6.jpg",
		},
		{
			Username:    "demosearch",
			Email:       "demo.search@aris.test",
			Phone:       "+79001000006",
			FirstName:   "Светлана",
			LastName:    "Поискова",
			Gender:      "female",
			Birthday:    "2001-06-15",
			Bio:         "Пользователь с редким именем для проверки backend-поиска.",
			Town:        "Пермь",
			Institution: "ПНИПУ",
			AvatarURL:   "https://i.ibb.co/s9rN3qD9/7.jpg",
		},
		{
			Username:    "demofresh",
			Email:       "demo.fresh@aris.test",
			Phone:       "+79001000007",
			FirstName:   "Антон",
			LastName:    "Свежесозданный",
			Gender:      "male",
			Birthday:    "2002-07-16",
			Bio:         "Новый тестовый пользователь с полностью заполненным профилем и аватаром.",
			Town:        "Ростов-на-Дону",
			Institution: "ЮФУ",
			AvatarURL:   "https://i.pravatar.cc/400?img=12",
		},
		{
			Username:    "demofeed",
			Email:       "demo.feed@aris.test",
			Phone:       "+79001000008",
			FirstName:   "Вера",
			LastName:    "Лентовая",
			Gender:      "female",
			Birthday:    "2000-08-17",
			Bio:         "Пользователь для проверки свежих постов, ленты и карточек профиля.",
			Town:        "Нижний Новгород",
			Institution: "ННГУ",
			AvatarURL:   "https://i.pravatar.cc/400?img=47",
		},
		{
			Username:    "demochat",
			Email:       "demo.chat@aris.test",
			Phone:       "+79001000009",
			FirstName:   "Павел",
			LastName:    "Диалогов",
			Gender:      "male",
			Birthday:    "1999-09-18",
			Bio:         "Пользователь для проверки личных сообщений и аватаров в чатах.",
			Town:        "Самара",
			Institution: "Самарский университет",
			AvatarURL:   "https://i.pravatar.cc/400?img=59",
		},
		{
			Username:    "demobackend",
			Email:       "demo.backend@aris.test",
			Phone:       "+79001000010",
			FirstName:   "Нина",
			LastName:    "Бэкендова",
			Gender:      "female",
			Birthday:    "2001-10-19",
			Bio:         "Пользователь с уникальными данными для проверки поиска друзей через backend.",
			Town:        "Томск",
			Institution: "ТГУ",
			AvatarURL:   "https://i.pravatar.cc/400?img=32",
		},
	}

	result := make(map[string]demoUserRecord, len(people))
	for _, person := range people {
		record, err := ensureDemoUser(ctx, db, person)
		if err != nil {
			return nil, err
		}
		result[person.Username] = record
	}

	return result, nil
}

func ensureDemoUser(ctx context.Context, db *pgxpool.Pool, user demoUser) (demoUserRecord, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return demoUserRecord{}, fmt.Errorf("hash demo password: %w", err)
	}

	var accountID int64
	err = db.QueryRow(ctx, `
		INSERT INTO user_account (uid, email, phone, password_hash, username, is_active)
		VALUES ($1, $2, $3, $4, $5, TRUE)
		ON CONFLICT (username) DO UPDATE
		SET email=EXCLUDED.email,
		    phone=EXCLUDED.phone,
		    password_hash=EXCLUDED.password_hash,
		    is_active=TRUE,
		    updated_at=NOW()
		RETURNING id
	`, uuid.New(), user.Email, user.Phone, string(passwordHash), strings.ToLower(user.Username)).Scan(&accountID)
	if err != nil {
		return demoUserRecord{}, fmt.Errorf("upsert user %s: %w", user.Username, err)
	}

	var profileID int64
	err = db.QueryRow(ctx, `
		SELECT p.id
		FROM profile p
		JOIN user_profile up ON up.profile_id=p.id
		WHERE up.user_account_id=$1
	`, accountID).Scan(&profileID)
	if err != nil {
		if err = db.QueryRow(ctx, `
			INSERT INTO profile (uid, is_active)
			VALUES ($1, TRUE)
			RETURNING id
		`, uuid.New()).Scan(&profileID); err != nil {
			return demoUserRecord{}, fmt.Errorf("create profile for %s: %w", user.Username, err)
		}
	}

	birthday, err := time.Parse("2006-01-02", user.Birthday)
	if err != nil {
		return demoUserRecord{}, err
	}

	nativeTown := user.NativeTown
	if nativeTown == "" {
		nativeTown = user.Town
	}
	studyGroup := user.StudyGroup
	if studyGroup == "" {
		studyGroup = "ARIS-DEMO"
	}
	company := user.Company
	if company == "" {
		company = "ARISNET Demo"
	}
	jobTitle := user.JobTitle
	if jobTitle == "" {
		jobTitle = "Участник тестового сценария"
	}
	interests := user.Interests
	if interests == "" {
		interests = "Социальные сети, тестирование интерфейсов, сообщества"
	}
	favMusic := user.FavMusic
	if favMusic == "" {
		favMusic = "Электроника, инди, lo-fi"
	}

	_, err = db.Exec(ctx, `
		INSERT INTO user_profile (
			uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date,
			gender, native_town, town, institution, study_group, company, job_title,
			interests, fav_music, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, TRUE)
		ON CONFLICT (user_account_id) DO UPDATE
		SET profile_id=EXCLUDED.profile_id,
		    first_name=EXCLUDED.first_name,
		    last_name=EXCLUDED.last_name,
		    bio=EXCLUDED.bio,
		    birthday_date=EXCLUDED.birthday_date,
		    gender=EXCLUDED.gender,
		    native_town=EXCLUDED.native_town,
		    town=EXCLUDED.town,
		    institution=EXCLUDED.institution,
		    study_group=EXCLUDED.study_group,
		    company=EXCLUDED.company,
		    job_title=EXCLUDED.job_title,
		    interests=EXCLUDED.interests,
		    fav_music=EXCLUDED.fav_music,
		    is_active=TRUE,
		    updated_at=NOW()
	`, uuid.New(), accountID, profileID, user.FirstName, user.LastName, user.Bio, birthday, user.Gender, nativeTown, user.Town, user.Institution, studyGroup, company, jobTitle, interests, favMusic)
	if err != nil {
		return demoUserRecord{}, fmt.Errorf("upsert user profile %s: %w", user.Username, err)
	}

	avatarID, err := ensureDemoMedia(ctx, db, "demo-avatar-"+user.Username, "jpg", "image/jpeg", user.AvatarURL, profileID)
	if err != nil {
		return demoUserRecord{}, err
	}
	if _, err = db.Exec(ctx, `UPDATE profile SET avatar_id=$1, is_active=TRUE, updated_at=NOW() WHERE id=$2`, avatarID, profileID); err != nil {
		return demoUserRecord{}, err
	}

	return demoUserRecord{AccountID: accountID, ProfileID: profileID}, nil
}

func seedDemoMedia(ctx context.Context, db *pgxpool.Pool, authorID int64) (map[string]int64, error) {
	items := map[string]string{
		"campus":    "https://moya-planeta.ru/upload/images/l/eb/e2/ebe21cb5a55a808b104f3d51c3ff96284bae5182.jpg",
		"frontend":  "https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=1200",
		"community": "https://images.unsplash.com/photo-1521737604893-d14cc237f11d?w=1200",
		"coffee":    "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?w=1200",
		"diagram":   "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=1200",
		"cover":     "https://images.unsplash.com/photo-1519389950473-47ba0277781c?w=1600",
	}

	result := make(map[string]int64, len(items))
	for key, link := range items {
		id, err := ensureDemoMedia(ctx, db, "demo-"+key, "jpg", "image/jpeg", link, authorID)
		if err != nil {
			return nil, err
		}
		result[key] = id
	}

	return result, nil
}

func ensureDemoMedia(ctx context.Context, db *pgxpool.Pool, name, extension, mimeType, link string, authorID int64) (int64, error) {
	var id int64
	err := db.QueryRow(ctx, `SELECT id FROM media WHERE media_name=$1 LIMIT 1`, name).Scan(&id)
	if err == nil {
		_, err = db.Exec(ctx, `
			UPDATE media
			SET extension=$1, mime_type=$2, link=$3, author_id=$4, is_active=TRUE, updated_at=NOW()
			WHERE id=$5
		`, extension, mimeType, link, authorID, id)
		return id, err
	}

	err = db.QueryRow(ctx, `
		INSERT INTO media (uid, media_name, extension, mime_type, size, link, author_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
		RETURNING id
	`, uuid.New(), name, extension, mimeType, int64(1), link, authorID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create media %s: %w", name, err)
	}
	return id, nil
}

func seedDemoFriendships(ctx context.Context, db *pgxpool.Pool, users map[string]demoUserRecord) error {
	pairs := []struct {
		Requester string
		Addressee string
		Status    string
	}{
		{"demoowner", "demoadmin", "accepted"},
		{"demoowner", "demomoderator", "accepted"},
		{"demoowner", "demomember", "accepted"},
		{"demoowner", "demofresh", "accepted"},
		{"demoowner", "demofeed", "accepted"},
		{"demoowner", "demochat", "accepted"},
		{"demosearch", "demoowner", "pending"},
		{"demobackend", "demoowner", "pending"},
		{"demoowner", "demoblocked", "pending"},
	}

	for _, pair := range pairs {
		if err := upsertFriendship(ctx, db, users[pair.Requester].ProfileID, users[pair.Addressee].ProfileID, pair.Status); err != nil {
			return err
		}
	}
	return nil
}

func upsertFriendship(ctx context.Context, db *pgxpool.Pool, requesterID, addresseeID int64, status string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO friendship (requester_id, addressee_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (requester_id, addressee_id) DO UPDATE
		SET status=EXCLUDED.status, updated_at=NOW()
	`, requesterID, addresseeID, status)
	return err
}

func seedDemoChats(ctx context.Context, db *pgxpool.Pool, users map[string]demoUserRecord) error {
	chatID, err := ensureDemoChat(ctx, db, "Демо чат: владелец и модератор", users["demoowner"].ProfileID, users["demomoderator"].ProfileID)
	if err != nil {
		return err
	}

	messages := []struct {
		Author string
		Text   string
		Age    time.Duration
	}{
		{"demoowner", "Привет! Этот чат нужен, чтобы проверить историю сообщений.", 45 * time.Minute},
		{"demomoderator", "Отлично, ещё проверим аватарки и порядок сообщений.", 38 * time.Minute},
		{"demoowner", "После отправки нового сообщения список должен обновиться без перезагрузки.", 12 * time.Minute},
	}

	for _, message := range messages {
		if err := ensureDemoMessage(ctx, db, chatID, users[message.Author].ProfileID, message.Text, time.Now().Add(-message.Age)); err != nil {
			return err
		}
	}

	freshChatID, err := ensureDemoChat(ctx, db, "Демо чат: владелец и новый пользователь", users["demoowner"].ProfileID, users["demochat"].ProfileID)
	if err != nil {
		return err
	}

	freshMessages := []struct {
		Author string
		Text   string
		Age    time.Duration
	}{
		{"demochat", "Привет! Я новый тестовый пользователь, у меня должен быть аватар в чате.", 18 * time.Minute},
		{"demoowner", "Отлично, проверим список диалогов, шапку чата и сообщения.", 15 * time.Minute},
		{"demochat", "Если всё заполнено в seed, заглушка с инициалами появляться не должна.", 6 * time.Minute},
	}

	for _, message := range freshMessages {
		if err := ensureDemoMessage(ctx, db, freshChatID, users[message.Author].ProfileID, message.Text, time.Now().Add(-message.Age)); err != nil {
			return err
		}
	}

	return nil
}

func ensureDemoChat(ctx context.Context, db *pgxpool.Pool, title string, profileIDs ...int64) (int64, error) {
	var id int64
	err := db.QueryRow(ctx, `SELECT id FROM chat WHERE title=$1 LIMIT 1`, title).Scan(&id)
	if err != nil {
		err = db.QueryRow(ctx, `
			INSERT INTO chat (uid, chat_type, title, is_active)
			VALUES ($1, 'personal', $2, TRUE)
			RETURNING id
		`, uuid.New(), title).Scan(&id)
		if err != nil {
			return 0, err
		}
	}

	for _, profileID := range profileIDs {
		_, err = db.Exec(ctx, `
			INSERT INTO chat_member (uid, chat_id, profile_id, chat_role, joined_at)
			VALUES ($1, $2, $3, 'member', NOW())
			ON CONFLICT (chat_id, profile_id) DO UPDATE
			SET leave_at=NULL, updated_at=NOW()
		`, uuid.New(), id, profileID)
		if err != nil {
			return 0, err
		}
	}

	return id, nil
}

func ensureDemoMessage(ctx context.Context, db *pgxpool.Pool, chatID, authorID int64, text string, createdAt time.Time) error {
	var exists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM message WHERE chat_id=$1 AND author_id=$2 AND message_text=$3)`, chatID, authorID, text).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err := db.Exec(ctx, `
		INSERT INTO message (uid, message_text, chat_id, author_id, status, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'sent', TRUE, $5, $5)
	`, uuid.New(), text, chatID, authorID, createdAt)
	return err
}

func seedDemoUserPosts(ctx context.Context, db *pgxpool.Pool, users map[string]demoUserRecord, media map[string]int64) error {
	posts := []struct {
		Key      string
		Author   string
		Text     string
		Age      time.Duration
		MediaIDs []int64
	}{
		{"fresh-user", "demoowner", "Свежий личный пост: его можно редактировать первые 10 минут.", 5 * time.Minute, []int64{media["campus"]}},
		{"old-user", "demoowner", "Старый личный пост: редактирование уже должно быть недоступно.", 25 * time.Minute, nil},
		{"five-images", "demomember", "Пост с пятью картинками для проверки сетки и просмотра фото.", 2 * time.Hour, []int64{media["campus"], media["frontend"], media["community"], media["coffee"], media["diagram"]}},
		{"fresh-profile-card", "demofresh", "Пост свежего тестового пользователя: аватар должен совпадать в профиле, друзьях, ленте и чате.", 8 * time.Minute, []int64{media["frontend"]}},
		{"fresh-feed-card", "demofeed", "Пост новой пользовательницы для проверки карточки ленты и backend-поиска по друзьям.", 55 * time.Minute, []int64{media["coffee"], media["diagram"]}},
	}

	for _, post := range posts {
		postID, err := ensureDemoPost(ctx, db, post.Key, users[post.Author].ProfileID, nil, post.Text, time.Now().Add(-post.Age), post.MediaIDs)
		if err != nil {
			return err
		}
		if err := ensureDemoLike(ctx, db, postID, users["demoadmin"].ProfileID); err != nil {
			return err
		}
		if err := ensureDemoComment(ctx, db, postID, users["demomoderator"].ProfileID, "Демо-комментарий для проверки счётчика."); err != nil {
			return err
		}
	}

	return nil
}

func seedDemoCommunities(ctx context.Context, db *pgxpool.Pool, users map[string]demoUserRecord, media map[string]int64) error {
	communityID, communityProfileID, err := ensureDemoCommunity(
		ctx,
		db,
		"demoaris",
		"ARIS Demo Community",
		"Сообщество для проверки ролей, постов от имени сообщества и поиска.",
		"public",
		media["community"],
		media["cover"],
	)
	if err != nil {
		return err
	}

	members := []struct {
		User string
		Role string
	}{
		{"demoowner", "owner"},
		{"demoadmin", "admin"},
		{"demomoderator", "moderator"},
		{"demomember", "member"},
		{"demofresh", "member"},
		{"demofeed", "member"},
		{"demoblocked", "blocked"},
	}
	for _, member := range members {
		if err := ensureDemoCommunityMember(ctx, db, communityID, users[member.User].ProfileID, member.Role); err != nil {
			return err
		}
	}

	communityPosts := []struct {
		Key      string
		AuthorID int64
		Text     string
		Age      time.Duration
		MediaIDs []int64
	}{
		{"community-fresh-official", communityProfileID, "Свежий пост от имени сообщества: модератор с правами может его редактировать.", 4 * time.Minute, []int64{media["community"]}},
		{"community-old-official", communityProfileID, "Старый пост от имени сообщества: редактирование после 10 минут должно быть закрыто.", 40 * time.Minute, nil},
		{"community-member-post", users["demomember"].ProfileID, "Пост участника внутри сообщества для проверки вкладок публикаций.", 70 * time.Minute, []int64{media["coffee"], media["diagram"]}},
	}

	for _, post := range communityPosts {
		if _, err := ensureDemoPost(ctx, db, post.Key, post.AuthorID, &communityID, post.Text, time.Now().Add(-post.Age), post.MediaIDs); err != nil {
			return err
		}
	}

	if _, _, err := ensureDemoCommunity(
		ctx,
		db,
		"searchclub",
		"Клуб backend поиска",
		"Сообщество с редким названием для проверки /api/search.",
		"public",
		media["frontend"],
		media["cover"],
	); err != nil {
		return err
	}

	return nil
}

func ensureDemoCommunity(
	ctx context.Context,
	db *pgxpool.Pool,
	username string,
	title string,
	bio string,
	communityType string,
	avatarID int64,
	coverID int64,
) (int64, int64, error) {
	var communityID int64
	var profileID int64
	err := db.QueryRow(ctx, `SELECT id, profile_id FROM community WHERE username=$1`, username).Scan(&communityID, &profileID)
	if err != nil {
		if err = db.QueryRow(ctx, `
			INSERT INTO profile (uid, avatar_id, is_active)
			VALUES ($1, $2, TRUE)
			RETURNING id
		`, uuid.New(), avatarID).Scan(&profileID); err != nil {
			return 0, 0, err
		}

		err = db.QueryRow(ctx, `
			INSERT INTO community (uid, title, bio, community_type, profile_id, username, cover_media_id, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
			RETURNING id
		`, uuid.New(), title, bio, communityType, profileID, username, coverID).Scan(&communityID)
		if err != nil {
			return 0, 0, err
		}
	}

	_, err = db.Exec(ctx, `
		UPDATE community
		SET title=$1, bio=$2, community_type=$3, cover_media_id=$4, is_active=TRUE, updated_at=NOW()
		WHERE id=$5
	`, title, bio, communityType, coverID, communityID)
	if err != nil {
		return 0, 0, err
	}

	_, err = db.Exec(ctx, `UPDATE profile SET avatar_id=$1, is_active=TRUE, updated_at=NOW() WHERE id=$2`, avatarID, profileID)
	if err != nil {
		return 0, 0, err
	}

	return communityID, profileID, nil
}

func ensureDemoCommunityMember(ctx context.Context, db *pgxpool.Pool, communityID, profileID int64, role string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO community_member (uid, profile_id, community_id, community_role, is_active, joined_at)
		VALUES ($1, $2, $3, $4, TRUE, NOW())
		ON CONFLICT (profile_id, community_id) DO UPDATE
		SET community_role=EXCLUDED.community_role, is_active=TRUE, leave_at=NULL, updated_at=NOW()
	`, uuid.New(), profileID, communityID, role)
	return err
}

func ensureDemoPost(
	ctx context.Context,
	db *pgxpool.Pool,
	key string,
	authorID int64,
	communityID *int64,
	text string,
	createdAt time.Time,
	mediaIDs []int64,
) (int64, error) {
	var postID int64
	err := db.QueryRow(ctx, `
		SELECT id
		FROM post
		WHERE author_id=$1
		  AND COALESCE(community_id, 0)=COALESCE($2, 0)
		  AND post_text=$3
		LIMIT 1
	`, authorID, communityID, text).Scan(&postID)
	if err != nil {
		err = db.QueryRow(ctx, `
			INSERT INTO post (uid, post_text, author_id, community_id, is_public_demo, allow_comments, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, FALSE, TRUE, TRUE, $5, $5)
			RETURNING id
		`, uuid.New(), text, authorID, communityID, createdAt).Scan(&postID)
		if err != nil {
			return 0, err
		}
	} else {
		_, err = db.Exec(ctx, `
			UPDATE post
			SET post_text=$1, author_id=$2, community_id=$3, is_public_demo=FALSE, is_active=TRUE, created_at=$4, updated_at=$4
			WHERE id=$5
		`, text, authorID, communityID, createdAt, postID)
		if err != nil {
			return 0, err
		}
	}

	if _, err = db.Exec(ctx, `DELETE FROM post_with_media WHERE post_id=$1`, postID); err != nil {
		return 0, err
	}
	for index, mediaID := range mediaIDs {
		if _, err = db.Exec(ctx, `
			INSERT INTO post_with_media (post_id, media_id, sort_order)
			VALUES ($1, $2, $3)
			ON CONFLICT (post_id, media_id) DO UPDATE SET sort_order=EXCLUDED.sort_order
		`, postID, mediaID, index); err != nil {
			return 0, err
		}
	}

	return postID, nil
}

func ensureDemoLike(ctx context.Context, db *pgxpool.Pool, postID, authorID int64) error {
	_, err := db.Exec(ctx, `
		INSERT INTO like_record (uid, post_id, author_id, is_active)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (post_id, author_id) WHERE post_id IS NOT NULL DO UPDATE
		SET is_active=TRUE, updated_at=NOW()
	`, uuid.New(), postID, authorID)
	return err
}

func ensureDemoComment(ctx context.Context, db *pgxpool.Pool, postID, authorID int64, text string) error {
	var exists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comment WHERE post_id=$1 AND author_id=$2 AND comment_text=$3)`, postID, authorID, text).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err := db.Exec(ctx, `
		INSERT INTO comment (uid, comment_text, post_id, author_id, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
	`, uuid.New(), text, postID, authorID)
	return err
}
