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
	chatGRPC "github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/handler/grpc"
	chatHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/handler/http"
	chatRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/repository"
	chatService "github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/service"
	appmetrics "github.com/go-park-mail-ru/2026_1_ARIS/internal/metrics"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	legacyChatRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat"
	chatMemberRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat_member"
	messageRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/message"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	sessionService "github.com/go-park-mail-ru/2026_1_ARIS/internal/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"
	chatpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/chat"
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
	logg.Info("logger initialized")

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

	userConn, err := grpc.NewClient(envString("USER_GRPC_ADDR", "localhost:8004"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect user grpc", zap.Error(err))
	}
	defer userConn.Close()

	store := chatRepo.NewStore(
		legacyChatRepo.NewChatStorage(db),
		chatMemberRepo.NewChatMemberStorage(db),
		messageRepo.NewMessageStorage(db),
	)
	chatSvc := chatService.New(store, userpb.NewUserServiceClient(userConn))
	hub := websocket.NewHub()
	go hub.Run()

	httpHandler := chatHTTP.New(chatSvc, hub)
	grpcHandler := chatGRPC.New(chatSvc)

	grpcServer := grpc.NewServer()
	chatpb.RegisterChatServiceServer(grpcServer, grpcHandler)
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", envInt("CHAT_GRPC_PORT", 8006)))
	if err != nil {
		logg.Fatal("failed to listen chat grpc", zap.Error(err))
	}
	go func() {
		logg.Info("chat grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve chat grpc", zap.Error(err))
		}
	}()

	sessions := sessionService.NewSessionService(sessionRepo.NewSessionStorage(redisClient))
	router := chi.NewRouter()
	router.Use(mymiddleware.RequestIDMiddleware(logg))
	router.Use(mymiddleware.AccessLogMiddleware(logg))
	router.Use(appmetrics.Middleware("chat"))
	router.Use(middleware.Recoverer)
	router.Handle("/metrics", appmetrics.Handler())
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, mymiddleware.AuthMiddleware(sessions))
	})
	httpHandler.RegisterWebSocketRoute(router, mymiddleware.AuthMiddleware(sessions))

	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", envInt("CHAT_HTTP_PORT", 8084)), Handler: router}
	go func() {
		logg.Info("chat http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve chat http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	grpcServer.GracefulStop()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("chat http server forced to shutdown", zap.Error(err))
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
