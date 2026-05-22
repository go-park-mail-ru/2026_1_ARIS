package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	gameHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/handler/http"
	gameMiddleware "github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/game/.env", ".env")

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

	authConn, err := grpc.NewClient(utils.EnvString("AUTH_GRPC_ADDR", "localhost:8002"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect auth grpc", zap.Error(err))
	}
	defer authConn.Close()

	gameUsecase := usecase.New(repository.NewStore(db), userpb.NewUserServiceClient(userConn))
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				if err := gameUsecase.CleanupEmptyWaitingRooms(cleanupCtx); err != nil {
					logg.Warn("failed to cleanup empty game rooms", zap.Error(err))
				}
			}
		}
	}()
	hub := websocket.NewHub()
	go hub.Run()

	httpHandler := gameHTTP.New(gameUsecase, hub)
	authMiddleware := gameMiddleware.AuthMiddleware(authpb.NewAuthServiceClient(authConn))

	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r, authMiddleware)
	})
	httpHandler.RegisterWebSocketRoute(router, authMiddleware)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("GAME_HTTP_PORT", 8089)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logg.Info("game http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve game http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("game service is stopping")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("game http server forced to shutdown", zap.Error(err))
	}
	logg.Info("game service stopped")
}
