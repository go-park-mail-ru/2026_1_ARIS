package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-park-mail-ru/2026_1_ARIS/docs"
	chathandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	authhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	feedhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/profile"
	userhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	memoryrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	chatrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
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
	chatservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/auth"
	mediaservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	postservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	sessionservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	connectdb "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// @title						ARIS backend API
// @version					1.0.0
// @description				Description of ARIS backend API
// @host						localhost:8080
// @BasePath					/api
// @accept					json
// @produce					json
// @schemes					http
// @securityDefinitions.apikey	SessionAuth
// @in							cookie
// @name						session_id
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment")
	}

	envConf, err := config.NewConfig()
	if err != nil {
		log.Fatal("can't get env config", err)
	}

	confStr, err := connectdb.GetConnectURL(envConf)
	if err != nil {
		log.Fatal("can't get db connection string", err)
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, confStr)
	if err != nil {
		log.Fatal("can't connect to db", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		log.Fatal("Bad db connection", err)
	}

	fmt.Println("Successfully connected to PostgreSQL")

	commentRepo := commentrepo.NewCommentStorage(db)
	repostRepo := repostrepo.NewRepostStorage(db)
	postRepo := postrepo.NewPostStorage(db)
	profileRepo := profilerepo.NewProfileStorage(db)
	likeRepo := likerepo.NewLikeStorage(db)
	userAccountRepo := useraccountrepo.NewUserAccountStorage(db)
	userProfileRepo := userprofilerepo.NewUserProfileStorage(db)
	sessionRepo := sessionrepo.NewSessionRepo()
	mediaRepo := mediarepo.NewMediaStorage(db)
	postWithMediaRepo := postrepo.NewPostWithMediaStorage(db)
	dbChatRepo := chatrepo.NewChatStorage(db)

	chatRepo := memoryrepo.NewChatRepo()
	chatMemberRepo := memoryrepo.NewChatMemberRepo()
	messageRepo := memoryrepo.NewMessageRepo()

	postService := postservice.NewPostService(postRepo, profileRepo, commentRepo, repostRepo, likeRepo)
	userService := userservice.NewUserService(userAccountRepo, profileRepo, userProfileRepo)
	authService := authservice.NewAuthService(userAccountRepo, profileRepo, userProfileRepo)
	sessService := sessionservice.NewSessionService(sessionRepo)
	mediaService := mediaservice.NewMediaService(mediaRepo, postWithMediaRepo)
	chatSvc := chatservice.NewChatService(chatRepo, chatMemberRepo, userService)
	messageSvc := chatservice.NewMessageService(messageRepo)

	authHandler := authhandler.NewAuthHandler(authService, sessService, userService)
	userHandler := &userhandler.UserHandler{
		UserService:  userService,
		MediaService: mediaService,
	}
	feedHandler := feedhandler.NewFeedHandler(postService, mediaService, userService)
	profileHandler := profile.NewProfileHandler(userService, mediaService, sessService)
	chatHandler := chathandler.NewChatHandler(chatSvc, messageSvc, userAccountRepo, userService)

	utils.MakeMock(mediaRepo, userService, postService, postWithMediaRepo, commentRepo, repostRepo, dbChatRepo)
	ensureKnownTestUser(ctx, userAccountRepo, userService, "sergeyshulginenko", "chatcheck123", "Сергей", "Шульгиненко")
	ensureKnownPassword(ctx, db, "sergeyshulginenko", "chatcheck123")
	seedDemoChats(ctx, userService, chatRepo, chatMemberRepo, messageRepo)
	seedGuaranteedChatForUsername(ctx, userAccountRepo, userService, chatRepo, chatMemberRepo, messageRepo, "ffffff", "sergeyshulginenko")

	fmt.Println("Swagger is running on http://localhost:8080/swagger/index.html")

	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler, profileHandler, chatHandler)

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

func ensureKnownPassword(ctx context.Context, db *pgxpool.Pool, username string, password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	_, _ = db.Exec(ctx, "UPDATE user_account SET password_hash=$1 WHERE username=$2", string(hash), username)
}

func ensureKnownTestUser(
	ctx context.Context,
	userAccountRepo useraccountrepo.UserAccountRepo,
	userService userservice.UserService,
	username string,
	password string,
	firstName string,
	lastName string,
) {
	if _, err := userAccountRepo.GetByUsername(ctx, username); err == nil {
		return
	}

	birthdayDate, err := time.Parse("02/01/2006", "24/02/2005")
	if err != nil {
		return
	}

	_, _ = userService.CreateRealUserProfile(
		ctx,
		nil,
		nil,
		password,
		username,
		firstName,
		lastName,
		birthdayDate,
		models.Gender("male"),
		nil,
	)
}

func seedGuaranteedChatForUsername(
	ctx context.Context,
	userAccountRepo useraccountrepo.UserAccountRepo,
	userService userservice.UserService,
	chatRepo memoryrepo.ChatRepo,
	chatMemberRepo memoryrepo.ChatMemberRepo,
	messageRepo memoryrepo.MessageRepo,
	username string,
	targetUsername string,
) {
	userAccount, err := userAccountRepo.GetByUsername(ctx, username)
	if err != nil {
		return
	}

	targetAccount, err := userAccountRepo.GetByUsername(ctx, targetUsername)
	if err != nil {
		return
	}

	_, err = userService.GetUserProfileByUserAccountID(ctx, userAccount.ID)
	if err != nil {
		return
	}

	targetProfile, err := userService.GetUserProfileByUserAccountID(ctx, targetAccount.ID)
	if err != nil {
		return
	}

	chat := models.NewChat(
		models.PrivateChat,
		fmt.Sprintf("%s %s", targetProfile.FirstName, targetProfile.LastName),
		nil,
	)
	if err := chatRepo.Save(ctx, *chat); err != nil {
		return
	}

	now := time.Now()
	members := []models.ChatMember{
		{
			ID:        rand.Int64(),
			Uid:       uuid.New(),
			ChatID:    chat.ID,
			MemberID:  userAccount.ID,
			JoinedAt:  now,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
			Role:      "member",
		},
		{
			ID:        rand.Int64(),
			Uid:       uuid.New(),
			ChatID:    chat.ID,
			MemberID:  targetAccount.ID,
			JoinedAt:  now,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
			Role:      "member",
		},
	}

	for _, member := range members {
		if err := chatMemberRepo.Save(ctx, member); err != nil {
			return
		}
	}

}

func seedDemoChats(
	ctx context.Context,
	userService userservice.UserService,
	chatRepo memoryrepo.ChatRepo,
	chatMemberRepo memoryrepo.ChatMemberRepo,
	messageRepo memoryrepo.MessageRepo,
) {
	users := userService.GetUserList(ctx, 0, 5)
	if len(users) < 2 {
		return
	}

	type pair struct {
		left  int
		right int
		texts []string
	}

	pairs := []pair{
		{left: 0, right: 1, texts: []string{"Привет! Как тебе новый интерфейс?", "Выглядит заметно лучше, особенно профиль.", "Супер, вечером добью ещё чаты."}},
		{left: 0, right: 2, texts: []string{"Ты сегодня на связи?", "Да, чуть позже отвечу подробнее."}},
		{left: 1, right: 3, texts: []string{"Скинь, пожалуйста, последние правки по макету.", "Уже отправил в общий чат."}},
	}

	for _, current := range pairs {
		if current.left >= len(users) || current.right >= len(users) {
			continue
		}

		leftUser := users[current.left]
		rightUser := users[current.right]
		leftProfile, err := userService.GetUserProfileByUserAccountID(ctx, leftUser.ID)
		if err != nil {
			continue
		}
		rightProfile, err := userService.GetUserProfileByUserAccountID(ctx, rightUser.ID)
		if err != nil {
			continue
		}

		chat := models.NewChat(
			models.PrivateChat,
			fmt.Sprintf("%s %s / %s %s", leftProfile.FirstName, leftProfile.LastName, rightProfile.FirstName, rightProfile.LastName),
			nil,
		)
		if err := chatRepo.Save(ctx, *chat); err != nil {
			continue
		}

		now := time.Now()
		members := []models.ChatMember{
			{
				ID:        rand.Int64(),
				Uid:       uuid.New(),
				ChatID:    chat.ID,
				MemberID:  leftUser.ID,
				JoinedAt:  now,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
				Role:      "member",
			},
			{
				ID:        rand.Int64(),
				Uid:       uuid.New(),
				ChatID:    chat.ID,
				MemberID:  rightUser.ID,
				JoinedAt:  now,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
				Role:      "member",
			},
		}

		for _, member := range members {
			_ = chatMemberRepo.Save(ctx, member)
		}

		for index, text := range current.texts {
			authorID := leftUser.ID
			if index%2 == 1 {
				authorID = rightUser.ID
			}
			createdAt := now.Add(time.Duration(index) * time.Minute)
			message := models.Message{
				ID:        rand.Int64(),
				Uid:       uuid.New(),
				Text:      &text,
				ChatID:    chat.ID,
				AuthorID:  authorID,
				IsActive:  true,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			}
			_ = messageRepo.Save(ctx, message)
		}
	}
}
