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
	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	postpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/post"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	postGRPC "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/handler/grpc"
	postHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/handler/http"
	postMiddleware "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/post/.env", ".env")

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

	postCache, err := tarantoolcache.InitTarantool(ctx)
	if err != nil {
		logg.Warn("tarantool post cache disabled", zap.Error(err))
	} else {
		defer postCache.Close()
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

	communityConn, err := grpc.NewClient(utils.EnvString("COMMUNITY_GRPC_ADDR", "localhost:8009"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect community grpc", zap.Error(err))
	}
	defer communityConn.Close()

	postUsecase := usecase.New(
		repository.NewStore(db),
		userpb.NewUserServiceClient(userConn),
		mediapb.NewMediaServiceClient(mediaConn),
		communitypb.NewCommunityServiceClient(communityConn),
	)
	if postCache != nil {
		postUsecase.SetCache(postCache)
	}

	grpcServer := grpc.NewServer()
	postpb.RegisterPostServiceServer(grpcServer, postGRPC.New(postUsecase))

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.EnvInt("POST_GRPC_PORT", 8005)))
	if err != nil {
		logg.Fatal("failed to listen post grpc", zap.Error(err))
	}

	go func() {
		logg.Info("post grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve post grpc", zap.Error(err))
		}
	}()

	httpHandler := postHTTP.New(postUsecase)
	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, postMiddleware.AuthMiddleware(authpb.NewAuthServiceClient(authConn)))
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("POST_HTTP_PORT", 8083)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logg.Info("post http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve post http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("post service is stopping")
	grpcServer.GracefulStop()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("post http server forced to shutdown", zap.Error(err))
	}
	logg.Info("post service stopped")
}
