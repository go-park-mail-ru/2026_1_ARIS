package auth

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

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

	authServise := authService.New(authRepo)

	GRPCAuthHandler := authHandler.New(authServise)

	authpb.RegisterAuthServiceServer(GRPCServer, GRPCAuthHandler)

	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", 8002))
		if err != nil {
			fmt.Println(err)
			logger.Error("failed to listen auth microservice", zap.Error(err))
		}
		if err := GRPCServer.Serve(listener); err != nil {
			fmt.Println(err)
			logger.Error("failed to serve grpc server", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Auth GRPC server is stopping...")
	logger.Info("Auth GRPC server is stopping...")

	GRPCServer.GracefulStop()

	fmt.Println("Auth GRPC server stopped")
	logger.Info("Auth GRPC server stopped")
}
