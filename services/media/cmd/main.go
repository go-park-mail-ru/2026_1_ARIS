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
	xminio "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	mediaGRPC "github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/handler/grpc"
	mediaHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/handler/http"
	mediaMiddleware "github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/media/.env", ".env")

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

	minioClient, err := xminio.New(ctx, logg)
	if err != nil {
		logg.Fatal("fail to connect MinIO", zap.Error(err))
	}

	userConn, err := grpc.NewClient(utils.EnvString("USER_GRPC_ADDR", "localhost:8004"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect user grpc", zap.Error(err))
	}
	defer userConn.Close()

	authConn, err := grpc.NewClient(utils.EnvString("AUTH_GRPC_ADDR", "localhost:8002"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect auth grpc", zap.Error(err))
	}
	defer authConn.Close()

	store := repository.NewStore(
		repository.NewMediaStorage(db),
		repository.NewMinioStorage(minioClient),
		utils.EnvString("MINIO_BUCKET_NAME", "media"),
	)
	mediaUsecase := usecase.New(store, userpb.NewUserServiceClient(userConn))
	httpHandler := mediaHTTP.New(mediaUsecase)

	grpcServer := grpc.NewServer()
	mediapb.RegisterMediaServiceServer(grpcServer, mediaGRPC.New(mediaUsecase))

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.EnvInt("MEDIA_GRPC_PORT", 8003)))
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
	router.Use(chimiddleware.Recoverer)
	metrics.RegisterHTTP(router, "media")
	router.Route("/api/media", func(r chi.Router) {
		r.Get("/{id}", httpHandler.RedirectToFile)
		r.Get("/{id}/url", httpHandler.GetFileURL)
		r.Group(func(r chi.Router) {
			r.Use(mediaMiddleware.AuthMiddleware(authpb.NewAuthServiceClient(authConn)))
			r.Post("/upload", httpHandler.SaveFiles)
		})
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("MEDIA_HTTP_PORT", 8081)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
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
