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
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	communityGRPC "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/handler/grpc"
	communityHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/handler/http"
	communityMiddleware "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/community/.env", ".env")

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

	userConn, err := grpc.NewClient(utils.EnvString("USER_GRPC_ADDR", "localhost:8004"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect user grpc", zap.Error(err))
	}
	defer userConn.Close()

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

	communityUsecase := usecase.New(
		repository.NewCommunityStorage(db),
		userpb.NewUserServiceClient(userConn),
		mediapb.NewMediaServiceClient(mediaConn),
	)

	grpcServer := grpc.NewServer()
	communitypb.RegisterCommunityServiceServer(grpcServer, communityGRPC.New(communityUsecase))

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.EnvInt("COMMUNITY_GRPC_PORT", 8009)))
	if err != nil {
		logg.Fatal("failed to listen community grpc", zap.Error(err))
	}

	go func() {
		logg.Info("community grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve community grpc", zap.Error(err))
		}
	}()

	httpHandler := communityHTTP.New(communityUsecase)
	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	metrics.RegisterHTTP(router, "community")
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, communityMiddleware.AuthMiddleware(authpb.NewAuthServiceClient(authConn)))
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("COMMUNITY_HTTP_PORT", 8087)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logg.Info("community http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve community http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("community service is stopping")
	grpcServer.GracefulStop()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("community http server forced to shutdown", zap.Error(err))
	}
	logg.Info("community service stopped")
}
