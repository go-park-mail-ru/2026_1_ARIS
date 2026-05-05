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
	appmetrics "github.com/go-park-mail-ru/2026_1_ARIS/internal/metrics"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	friendrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend"
	mediarepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	profilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	sessionrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	settingsrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings"
	useraccountrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofilerepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/session"
	supportgrpc "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/handler/grpc"
	supporthttp "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/handler/http"
	supportrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/repository"
	supportservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/service"
	userrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/user/repository"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/user/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	redisClient, err := redis.InitRedis(ctx, envConf)
	if err != nil {
		logg.Fatal("fail to connect Redis", zap.Error(err))
	}

	mediaConn, err := grpc.NewClient(envString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	userStore := userrepo.NewStore(
		useraccountrepo.NewUserAccountStorage(db),
		profilerepo.NewProfileStorage(db),
		userprofilerepo.NewUserProfileStorage(db),
		settingsrepo.NewUserSettingsStorage(db),
		friendrepo.NewFriendshipStorage(db),
	)
	userSvc := userservice.New(userStore, mediapb.NewMediaServiceClient(mediaConn))
	ticketSvc := supportservice.NewTicketService(
		supportrepo.NewTicketStorage(db),
		mediarepo.NewMediaStorage(db),
		mediapb.NewMediaServiceClient(mediaConn),
	)
	seedSupportAdmins(ctx, logg, ticketSvc, 1, 2, 3, 4)

	grpcServer := grpc.NewServer()
	supportpb.RegisterSupportServiceServer(grpcServer, supportgrpc.New(ticketSvc))
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", envInt("SUPPORT_GRPC_PORT", 8007)))
	if err != nil {
		logg.Fatal("failed to listen support grpc", zap.Error(err))
	}
	go func() {
		logg.Info("support grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve support grpc", zap.Error(err))
		}
	}()

	sessions := session.NewSessionService(sessionrepo.NewSessionStorage(redisClient))
	hub := websocket.NewHub()
	go hub.Run()

	httpHandler := supporthttp.NewSupportHandler(sessions, userSvc, ticketSvc, hub)
	router := chi.NewRouter()
	router.Use(mymiddleware.RequestIDMiddleware(logg))
	router.Use(mymiddleware.AccessLogMiddleware(logg))
	router.Use(appmetrics.Middleware("support"))
	router.Use(middleware.Recoverer)
	router.Use(mymiddleware.XSSMiddlewares()...)
	router.Handle("/metrics", appmetrics.Handler())
	router.Route("/api/support", func(r chi.Router) {
		r.Use(mymiddleware.AuthMiddleware(sessions))
		httpHandler.RegisterRoutes(r)
	})
	router.Group(func(r chi.Router) {
		r.Use(mymiddleware.AuthMiddleware(sessions))
		r.Get("/ws/support/{ticketID}", httpHandler.HandleTicketWebSocket)
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", envInt("SUPPORT_HTTP_PORT", 8086)),
		Handler: router,
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

func seedSupportAdmins(ctx context.Context, logg *zap.Logger, ticketSvc supportservice.TicketService, profileIDs ...int64) {
	for _, profileID := range profileIDs {
		if err := supportservice.MakeProfileAdmin(ctx, ticketSvc, profileID); err != nil {
			logg.Warn("failed to seed support admin", zap.Int64("profile_id", profileID), zap.Error(err))
		}
	}
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
