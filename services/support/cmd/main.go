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
	mediarepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	supportGRPC "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/handler/grpc"
	supportHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/handler/http"
	supportrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/repository"
	supportsvc "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/support/.env", ".env")

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

	authConn, err := grpc.NewClient(utils.EnvString("AUTH_GRPC_ADDR", "localhost:8002"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect auth grpc", zap.Error(err))
	}
	defer authConn.Close()

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

	ticketService := supportsvc.NewTicketService(
		supportrepo.NewTicketStorage(db),
		mediarepo.NewMediaStorage(db),
		mediapb.NewMediaServiceClient(mediaConn),
	)
	seedSupportAdmins(ctx, logg, ticketService, 1, 2, 3, 4)

	grpcServer := grpc.NewServer()
	supportpb.RegisterSupportServiceServer(grpcServer, supportGRPC.New(ticketService))

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.EnvInt("SUPPORT_GRPC_PORT", 8007)))
	if err != nil {
		logg.Fatal("failed to listen support grpc", zap.Error(err))
	}

	go func() {
		logg.Info("support grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve support grpc", zap.Error(err))
		}
	}()

	hub := websocket.NewHub()
	go hub.Run()

	httpHandler := supportHTTP.NewSupportHandler(
		authSessionService{client: authpb.NewAuthServiceClient(authConn)},
		grpcUserService{client: userpb.NewUserServiceClient(userConn)},
		ticketService,
		hub,
	)
	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Route("/api/support", func(r chi.Router) {
		httpHandler.RegisterRoutes(r)
	})
	router.Route("/ws/support", func(r chi.Router) {
		r.Get("/tickets/{ticketID}", httpHandler.HandleTicketWebSocket)
		r.Get("/{ticketID}", httpHandler.HandleTicketWebSocket)
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("SUPPORT_HTTP_PORT", 8086)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logg.Info("support http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve support http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("support service is stopping")
	grpcServer.GracefulStop()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("support http server forced to shutdown", zap.Error(err))
	}

	logg.Info("support service stopped")
}

func seedSupportAdmins(ctx context.Context, logg *zap.Logger, service supportsvc.TicketService, profileIDs ...int64) {
	for _, profileID := range profileIDs {
		if err := supportsvc.MakeProfileAdmin(ctx, service, profileID); err != nil {
			logg.Warn("failed to seed support admin", zap.Int64("profile_id", profileID), zap.Error(err))
		}
	}
}
