package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	friendRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend"
	profileRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	settingsRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings"
	accountRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userProfileRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	sessionService "github.com/go-park-mail-ru/2026_1_ARIS/internal/session"
	userGRPC "github.com/go-park-mail-ru/2026_1_ARIS/internal/user/handler/grpc"
	userHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/user/handler/http"
	userRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/user/repository"
	userService "github.com/go-park-mail-ru/2026_1_ARIS/internal/user/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment: ", err)
	}

	ctx := context.Background()

	logg, err := logger.New()
	if err != nil {
		log.Fatal("fail to create logger: ", err)
	}
	defer func() {
		if err := logg.Sync(); err != nil {
			logg.Error("fail to sync logger", zap.Error(err))
		}
	}()

	envConf, err := config.NewConfig()
	if err != nil {
		logg.Fatal("fail to load env variables", zap.Error(err))
	}

	db, err := postgres.New(ctx, envConf)
	if err != nil {
		logg.Fatal("fail to connect PostgreSQL", zap.Error(err))
	}

	redisClient, err := redis.InitRedis(ctx, envConf)
	if err != nil {
		logg.Fatal("fail to connect Redis", zap.Error(err))
	}

	mediaConn, err := grpc.NewClient(envString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	store := userRepo.NewStore(
		accountRepo.NewUserAccountStorage(db),
		profileRepo.NewProfileStorage(db),
		userProfileRepo.NewUserProfileStorage(db),
		settingsRepo.NewUserSettingsStorage(db),
		friendRepo.NewFriendshipStorage(db),
	)
	userSvc := userService.New(store, mediapb.NewMediaServiceClient(mediaConn))
	httpHandler := userHTTP.New(userSvc)
	grpcHandler := userGRPC.New(userSvc)

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, grpcHandler)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", envInt("USER_GRPC_PORT", 8004)))
	if err != nil {
		logg.Fatal("failed to listen user grpc", zap.Error(err))
	}

	go func() {
		logg.Info("user grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve user grpc", zap.Error(err))
		}
	}()

	sessions := sessionService.NewSessionService(sessionRepo.NewSessionStorage(redisClient))
	router := chi.NewRouter()
	router.Use(mymiddleware.RequestIDMiddleware(logg))
	router.Use(mymiddleware.AccessLogMiddleware(logg))
	router.Use(middleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, mymiddleware.AuthMiddleware(sessions))
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", envInt("USER_HTTP_PORT", 8082)),
		Handler: router,
	}

	go func() {
		logg.Info("user http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve user http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("user service is stopping")
	grpcServer.GracefulStop()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("user http server forced to shutdown", zap.Error(err))
	}

	logg.Info("user service stopped")
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envString(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
