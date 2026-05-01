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
	authhandler "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/auth"
	appmiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	profileRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	userAccountRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userProfileRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/auth"
	sessionservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	logg.Info("Successfully connected to PostgreSQL")

	redisClient, err := redis.InitRedis(ctx, envConf)
	if err != nil {
		logg.Fatal("fail to connect Redis", zap.Error(err))
	}
	logg.Info("Successfully connected to Redis")

	profileStore := profileRepo.NewProfileStorage(db)
	sessionStore := sessionRepo.NewSessionStorage(redisClient)
	userAccountStore := userAccountRepo.NewUserAccountStorage(db)
	userProfileStore := userProfileRepo.NewUserProfileStorage(db)

	authSvc := authservice.NewAuthService(userAccountStore, profileStore, userProfileStore)
	sessionSvc := sessionservice.NewSessionService(sessionStore)
	userSvc := userservice.NewUserService(userAccountStore, profileStore, userProfileStore)
	authHandler := authhandler.NewAuthHandler(authSvc, sessionSvc, userSvc)

	grpcServer := grpc.NewServer()
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", envInt("AUTH_GRPC_PORT", 8002)))
	if err != nil {
		logg.Fatal("failed to listen auth grpc", zap.Error(err))
	}

	go func() {
		logg.Info("auth grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve grpc server", zap.Error(err))
		}
	}()

	router := chi.NewRouter()
	router.Use(appmiddleware.RequestIDMiddleware(logg))
	router.Use(appmiddleware.AccessLogMiddleware(logg))
	router.Use(middleware.Recoverer)
	router.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/register/step-one", authHandler.ValidateRegisterStepOne)
		r.Post("/login", authHandler.Login)
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.AuthMiddleware(sessionSvc))
			r.Get("/me", authHandler.Me)
			r.Post("/logout", authHandler.Logout)
		})
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", envInt("AUTH_HTTP_PORT", 8085)),
		Handler: router,
	}

	go func() {
		logg.Info("auth http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve auth http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("auth service is stopping")

	grpcServer.GracefulStop()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("auth http server forced to shutdown", zap.Error(err))
	}

	logg.Info("auth service stopped")
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
