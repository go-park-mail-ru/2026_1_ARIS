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
	chatpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/chat"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	chatGRPC "github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/handler/grpc"
	chatHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/handler/http"
	chatMiddleware "github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/chat/.env", ".env")

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

	presenceCache, err := tarantoolcache.InitTarantool(ctx)
	if err != nil {
		logg.Warn("tarantool presence cache disabled", zap.Error(err))
	} else {
		defer presenceCache.Close()
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

	mediaConn, err := grpc.NewClient(utils.EnvString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	chatUsecase := usecase.New(repository.NewStore(db), userpb.NewUserServiceClient(userConn), mediapb.NewMediaServiceClient(mediaConn))
	var hub *websocket.Hub
	if presenceCache != nil {
		chatUsecase.SetPresenceReader(presenceCache)
		hub = websocket.NewHub(presenceCache)
	} else {
		hub = websocket.NewHub()
	}
	go hub.Run()

	grpcServer := grpc.NewServer()
	chatpb.RegisterChatServiceServer(grpcServer, chatGRPC.New(chatUsecase))

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.EnvInt("CHAT_GRPC_PORT", 8006)))
	if err != nil {
		logg.Fatal("failed to listen chat grpc", zap.Error(err))
	}
	go func() {
		logg.Info("chat grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve chat grpc", zap.Error(err))
		}
	}()

	httpHandler := chatHTTP.New(chatUsecase, hub)
	authMiddleware := chatMiddleware.AuthMiddleware(authpb.NewAuthServiceClient(authConn))
	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, authMiddleware)
	})
	httpHandler.RegisterWebSocketRoute(router, authMiddleware)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("CHAT_HTTP_PORT", 8084)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logg.Info("chat http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve chat http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("chat service is stopping")
	grpcServer.GracefulStop()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("chat http server forced to shutdown", zap.Error(err))
	}
	logg.Info("chat service stopped")
}
