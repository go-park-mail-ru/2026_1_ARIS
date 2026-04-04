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

	chathandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	wsHandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler"
	authhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	feedhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/feed"
	friendshiphandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/friend"
	profilehandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/profile"
	userhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	chatstorage "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	commentrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	friendshiprepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend"
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
	friendshipservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/friend"
	mediaservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	postservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	sessionservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/config"
	connectdb "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/connect_db"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

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
	friendshipRepo := friendshiprepo.NewFriendshipStorage(db)

	chatRepo := chatstorage.NewSQLChatRepo(db)
	chatMemberRepo := repository.NewChatMemberStorage(db)
	messageRepo := repository.NewMessageStorage(db)

	postService := postservice.NewPostService(postRepo, profileRepo, commentRepo, repostRepo, likeRepo)
	userService := userservice.NewUserService(userAccountRepo, profileRepo, userProfileRepo)
	authService := authservice.NewAuthService(userAccountRepo, profileRepo, userProfileRepo)
	sessService := sessionservice.NewSessionService(sessionRepo)
	mediaService := mediaservice.NewMediaService(mediaRepo, postWithMediaRepo)
	chatSvc := chatservice.NewChatService(chatRepo, chatMemberRepo, userService)
	messageSvc := chatservice.NewMessageService(messageRepo)
	friendshipService := friendshipservice.NewFriendshipService(friendshipRepo)

	// WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	authHandler := authhandler.NewAuthHandler(authService, sessService, userService)
	userHandler := userhandler.NewUserHandler(userService, mediaService)
	feedHandler := feedhandler.NewFeedHandler(postService, mediaService, userService)
	profileHandler := profilehandler.NewProfileHandler(userService, mediaService, sessService)
	chatHandler := chathandler.NewChatHandler(chatSvc, messageSvc, userAccountRepo, userService, hub)
	friendHandler := friendshiphandler.NewFriendHandler(sessService, userService, friendshipService)
	wsHandler := wsHandler.NewWebSocketHandler(hub, chatSvc)

	router := server.NewRouter(authHandler, sessService, feedHandler, userHandler, profileHandler, chatHandler, friendHandler, wsHandler)

	utils.MakeMock(mediaRepo, userService, postService, postWithMediaRepo, commentRepo, repostRepo, chatRepo)

	ensureKnownTestUser(ctx, userAccountRepo, userService, "sergeyshulginenko", "chatcheck123", "Сергей", "Шульгиненко")
	ensureKnownPassword(ctx, db, "sergeyshulginenko", "chatcheck123")

	fmt.Println("Swagger is running on http://localhost:8080/swagger/index.html")

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

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
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
