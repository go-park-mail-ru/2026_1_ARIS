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
	authHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/handler/http"
	profileRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	userAccountRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userProfileRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"

	authHandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/handler/grpc"
	authRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/repository"
	authService "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/service"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment: ", err)
	}

	ctx := context.Background()

	// Создаём логгер
	logger, err := logger.New()
	if err != nil {
		log.Fatal("fail to create logger: ", err)
	}
	logger.Info("logger initialized")

	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("fail to sync logger", zap.Error(err))
		}
	}()

	// Создаём конфиг
	envConf, err := config.NewConfig()
	if err != nil {
		logger.Fatal("fail to load env variables", zap.Error(err))
	}
	logger.Info("Config parsed")

	// Подключаем к БД
	db, err := postgres.New(ctx, envConf)
	if err != nil {
		logger.Fatal("fail to connect PostgreSQL", zap.Error(err))
	}
	logger.Info("Successfully connected to PostgreSQL")

	// Подключаем Redis
	redisClient, err := redis.InitRedis(ctx, envConf)
	if err != nil {
		logger.Fatal("fail to connect Redis", zap.Error(err))
	}
	logger.Info("Successfully connected to Redis")

	GRPCServer := grpc.NewServer()

	profileRepo := profileRepo.NewProfileStorage(db)
	sessionRepo := sessionRepo.NewSessionStorage(redisClient)
	userAccountRepo := userAccountRepo.NewUserAccountStorage(db)
	userProfileRepo := userProfileRepo.NewUserProfileStorage(db)

	authRepo := authRepo.NewStore(userAccountRepo, profileRepo, userProfileRepo, sessionRepo)

	mediaConn, err := grpc.NewClient(envString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	authServise := authService.New(authRepo, mediapb.NewMediaServiceClient(mediaConn))

	GRPCAuthHandler := authHandler.New(authServise)

	authpb.RegisterAuthServiceServer(GRPCServer, GRPCAuthHandler)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", envInt("AUTH_GRPC_PORT", 8002)))
	if err != nil {
		logger.Fatal("failed to listen auth grpc", zap.Error(err))
	}

	go func() {
		logger.Info("auth grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := GRPCServer.Serve(grpcListener); err != nil {
			fmt.Println(err)
			logger.Error("failed to serve grpc server", zap.Error(err))
		}
	}()

	httpHandler := authHTTP.New(authServise, false)
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Route("/api/auth", func(r chi.Router) {
		httpHandler.RegisterRoutes(r)
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", envInt("AUTH_HTTP_PORT", 8085)),
		Handler: router,
	}

	go func() {
		logger.Info("auth http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to serve auth http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Auth service is stopping...")
	logger.Info("Auth service is stopping...")

	GRPCServer.GracefulStop()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logger.Fatal("auth http server forced to shutdown", zap.Error(err))
	}

	fmt.Println("Auth service stopped")
	logger.Info("Auth service stopped")
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
