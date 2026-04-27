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
	mediaGRPC "github.com/go-park-mail-ru/2026_1_ARIS/internal/media/handler/grpc"
	mediaHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/media/handler/http"
	mediaRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/media/repository"
	mediaService "github.com/go-park-mail-ru/2026_1_ARIS/internal/media/service"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	legacyMediaRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	sessionService "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
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

	minioClient, err := xminio.New(ctx, envConf, logg)
	if err != nil {
		logg.Fatal("fail to connect MinIO", zap.Error(err))
	}

	mediaStorage := legacyMediaRepo.NewMediaStorage(db)
	s3 := legacyMediaRepo.NewMinioClient(minioClient)
	sessions := sessionService.NewSessionService(sessionRepo.NewSessionStorage(redisClient))

	userConn, err := grpc.NewClient(envString("USER_GRPC_ADDR", "localhost:8004"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect user grpc", zap.Error(err))
	}
	defer userConn.Close()

	userClient := userpb.NewUserServiceClient(userConn)

	store := mediaRepo.NewStore(mediaStorage, s3, envConf.MinioBucketName)
	mediaSvc := mediaService.New(store, userClient)
	httpHandler := mediaHTTP.New(mediaSvc)
	grpcHandler := mediaGRPC.New(mediaSvc)

	grpcServer := grpc.NewServer()
	mediapb.RegisterMediaServiceServer(grpcServer, grpcHandler)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", envInt("MEDIA_GRPC_PORT", 8003)))
	if err != nil {
		logg.Fatal("failed to listen media grpc", zap.Error(err))
	}

	go func() {
		logg.Info("media grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve media grpc", zap.Error(err))
		}
	}()

	router := chi.NewRouter()
	router.Use(mymiddleware.RequestIDMiddleware(logg))
	router.Use(mymiddleware.AccessLogMiddleware(logg))
	router.Use(middleware.Recoverer)
	router.Route("/api/media", func(r chi.Router) {
		r.Get("/{id}", httpHandler.RedirectToFile)
		r.Get("/{id}/url", httpHandler.GetFileURL)
		r.Group(func(r chi.Router) {
			r.Use(mymiddleware.AuthMiddleware(sessions))
			r.Post("/upload", httpHandler.SaveFiles)
		})
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", envInt("MEDIA_HTTP_PORT", 8081)),
		Handler: router,
	}

	go func() {
		logg.Info("media http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve media http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("media service is stopping")
	grpcServer.GracefulStop()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("media http server forced to shutdown", zap.Error(err))
	}

	logg.Info("media service stopped")
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
