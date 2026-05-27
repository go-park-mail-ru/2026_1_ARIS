package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/redis"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	authGRPC "github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/handler/grpc"
	authHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/handler/http"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type supportRoleProvider struct {
	client supportpb.SupportServiceClient
}

func (p supportRoleProvider) GetProfileRole(ctx context.Context, profileID int64) (model.SupportRole, error) {
	resp, err := p.client.GetProfileRole(ctx, &supportpb.GetProfileRoleRequest{ProfileId: profileID})
	if err != nil {
		return model.SupportRoleUser, err
	}
	return model.SupportRole(resp.GetRole()), nil
}

func main() {
	_ = godotenv.Load("services/auth/.env", ".env")

	ctx := context.Background()

	logg, err := logger.New(utils.EnvString("LOGGER_LEVEL", "info"))
	if err != nil {
		log.Fatal("fail to create logger: ", err)
	}
	defer func() {
		if err := logg.Sync(); err != nil {
			logg.Error("fail to sync logger", zap.Error(err))
		}
	}()

	redisClient, err := redis.InitRedis(ctx)
	if err != nil {
		logg.Fatal("fail to connect Redis", zap.Error(err))
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

	supportConn, err := grpc.NewClient(utils.EnvString("SUPPORT_GRPC_ADDR", "localhost:8007"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect support grpc", zap.Error(err))
	}
	defer supportConn.Close()

	appEndpoint := strings.TrimRight(utils.EnvString("APP_ENDPOINT", "https://arisnet.ru"), "/")
	vkidRedirectURI := utils.EnvString("VKID_REDIRECT_URI", appEndpoint+"/api/auth/vkid/callback")
	authUsecase := usecase.New(
		repository.NewSessionStorage(redisClient),
		userpb.NewUserServiceClient(userConn),
		mediapb.NewMediaServiceClient(mediaConn),
		usecase.WithVKIDClient(usecase.NewVKIDHTTPClient(usecase.VKIDConfig{
			ClientID:     utils.EnvString("VKID_CLIENT_ID", ""),
			ClientSecret: utils.EnvString("VKID_CLIENT_SECRET", ""),
			TokenURL:     utils.EnvString("VKID_TOKEN_URL", ""),
			UserInfoURL:  utils.EnvString("VKID_USER_INFO_URL", ""),
			UsersGetURL:  utils.EnvString("VKID_USERS_GET_URL", ""),
		})),
	)

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, authGRPC.New(authUsecase))

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", utils.EnvInt("AUTH_GRPC_PORT", 8002)))
	if err != nil {
		logg.Fatal("failed to listen auth grpc", zap.Error(err))
	}

	go func() {
		logg.Info("auth grpc server started", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("failed to serve auth grpc", zap.Error(err))
		}
	}()

	httpHandler := authHTTP.New(authUsecase, false, supportRoleProvider{client: supportpb.NewSupportServiceClient(supportConn)})
	httpHandler.ConfigureOAuthStates(repository.NewOAuthStateStorage(redisClient))
	httpHandler.ConfigureVKID(authHTTP.VKIDConfig{
		ClientID:            utils.EnvString("VKID_CLIENT_ID", ""),
		AuthorizeURL:        utils.EnvString("VKID_AUTHORIZE_URL", ""),
		RedirectURI:         vkidRedirectURI,
		Scope:               utils.EnvString("VKID_SCOPE", "email"),
		FrontendSuccessPath: utils.EnvString("VKID_FRONTEND_SUCCESS_PATH", "/feed?oauth=vkid"),
		FrontendErrorPath:   utils.EnvString("VKID_FRONTEND_ERROR_PATH", "/login"),
	})
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	metrics.RegisterHTTP(router, "auth")
	router.Route("/api/auth", func(r chi.Router) {
		httpHandler.RegisterRoutes(r)
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("AUTH_HTTP_PORT", 8085)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
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
