package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	communityHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/community/handler/http"
	communityRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/community/repository"
	communityService "github.com/go-park-mail-ru/2026_1_ARIS/internal/community/service"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	legacyCommunityRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	sessionService "github.com/go-park-mail-ru/2026_1_ARIS/internal/session"
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
	defer func() { _ = logg.Sync() }()

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

	userConn, err := grpc.NewClient(envString("USER_GRPC_ADDR", "localhost:8004"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect user grpc", zap.Error(err))
	}
	defer userConn.Close()
	mediaConn, err := grpc.NewClient(envString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	store := communityRepo.NewStore(legacyCommunityRepo.NewCommunityStorage(db))
	communitySvc := communityService.New(store, userpb.NewUserServiceClient(userConn), mediapb.NewMediaServiceClient(mediaConn))
	httpHandler := communityHTTP.New(communitySvc)

	sessions := sessionService.NewSessionService(sessionRepo.NewSessionStorage(redisClient))
	router := chi.NewRouter()
	router.Use(mymiddleware.RequestIDMiddleware(logg))
	router.Use(mymiddleware.AccessLogMiddleware(logg))
	router.Use(middleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, mymiddleware.AuthMiddleware(sessions))
	})

	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", envInt("COMMUNITY_HTTP_PORT", 8087)), Handler: router}
	go func() {
		logg.Info("community http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve community http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("community http server forced to shutdown", zap.Error(err))
	}
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
