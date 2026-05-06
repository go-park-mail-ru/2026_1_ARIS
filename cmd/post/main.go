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
	appmetrics "github.com/go-park-mail-ru/2026_1_ARIS/internal/metrics"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	postGRPC "github.com/go-park-mail-ru/2026_1_ARIS/internal/post/handler/grpc"
	postHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/post/handler/http"
	postRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/post/repository"
	postService "github.com/go-park-mail-ru/2026_1_ARIS/internal/post/service"
	commentRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	communityRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community"
	likeRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	legacyPostRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	repostRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	sessionService "github.com/go-park-mail-ru/2026_1_ARIS/internal/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	postpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/post"
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

	store := postRepo.NewStore(
		legacyPostRepo.NewPostStorage(db),
		legacyPostRepo.NewPostWithMediaStorage(db),
		commentRepo.NewCommentStorage(db),
		likeRepo.NewLikeStorage(db),
		repostRepo.NewRepostStorage(db),
		communityRepo.NewCommunityStorage(db),
	)
	postSvc := postService.New(store, userpb.NewUserServiceClient(userConn), mediapb.NewMediaServiceClient(mediaConn))
	httpHandler := postHTTP.New(postSvc)
	grpcHandler := postGRPC.New(postSvc)

	grpcServer := grpc.NewServer()
	postpb.RegisterPostServiceServer(grpcServer, grpcHandler)
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", envInt("POST_GRPC_PORT", 8005)))
	if err != nil {
		logg.Fatal("failed to listen post grpc", zap.Error(err))
	}
	go func() {
		logg.Info("post grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve post grpc", zap.Error(err))
		}
	}()

	sessions := sessionService.NewSessionService(sessionRepo.NewSessionStorage(redisClient))
	router := chi.NewRouter()
	router.Use(mymiddleware.RequestIDMiddleware(logg))
	router.Use(mymiddleware.AccessLogMiddleware(logg))
	router.Use(appmetrics.Middleware("post"))
	router.Use(middleware.Recoverer)
	router.Handle("/metrics", appmetrics.Handler())
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, mymiddleware.AuthMiddleware(sessions))
	})
	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", envInt("POST_HTTP_PORT", 8083)), Handler: router}
	go func() {
		logg.Info("post http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve post http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	grpcServer.GracefulStop()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("post http server forced to shutdown", zap.Error(err))
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
