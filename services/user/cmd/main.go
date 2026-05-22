package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	tarantoolcache "github.com/go-park-mail-ru/2026_1_ARIS/pkg/tarantool"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	userGRPC "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/handler/grpc"
	userHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/handler/http"
	userMiddleware "github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/user/.env", ".env")

	ctx := context.Background()

	logg, err := logger.New("info")
	if err != nil {
		log.Fatal("fail to create logger: ", err)
	}
	defer func() {
		if err := logg.Sync(); err != nil {
			logg.Error("fail to sync logger", zap.Error(err))
		}
	}()

	db, err := postgres.New(ctx)
	if err != nil {
		logg.Fatal("fail to connect PostgreSQL", zap.Error(err))
	}

	profileCache, err := tarantoolcache.InitTarantool(ctx)
	if err != nil {
		logg.Warn("tarantool profile cache disabled", zap.Error(err))
	} else {
		defer profileCache.Close()
	}

	mediaConn, err := grpc.NewClient(utils.EnvString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	authConn, err := grpc.NewClient(utils.EnvString("AUTH_GRPC_ADDR", "localhost:8002"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect auth grpc", zap.Error(err))
	}
	defer authConn.Close()

	userUsecase := usecase.New(repository.NewStore(db), mediapb.NewMediaServiceClient(mediaConn))
	if profileCache != nil {
		userUsecase.SetCache(profileCache)
	}

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, userGRPC.New(userUsecase))

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.EnvInt("USER_GRPC_PORT", 8004)))
	if err != nil {
		logg.Fatal("failed to listen user grpc", zap.Error(err))
	}

	go func() {
		logg.Info("user grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve user grpc", zap.Error(err))
		}
	}()

	httpHandler := userHTTP.New(userUsecase)
	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, userMiddleware.AuthMiddleware(authpb.NewAuthServiceClient(authConn)))
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("USER_HTTP_PORT", 8082)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
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
