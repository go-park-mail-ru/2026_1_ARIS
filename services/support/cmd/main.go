package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	supportGRPC "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/handler/grpc"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

	supportUsecase := usecase.New(repository.NewProfileRoleStorage(db))
	seedSupportAdmins(ctx, logg, supportUsecase, 1, 2, 3, 4)

	grpcServer := grpc.NewServer()
	supportpb.RegisterSupportServiceServer(grpcServer, supportGRPC.New(supportUsecase))

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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("support service is stopping")
	grpcServer.GracefulStop()
	logg.Info("support service stopped")
}

func seedSupportAdmins(ctx context.Context, logg *zap.Logger, service *usecase.Service, profileIDs ...int64) {
	for _, profileID := range profileIDs {
		if err := service.SetProfileRole(ctx, profileID, model.SupportRoleAdmin); err != nil {
			logg.Warn("failed to seed support admin", zap.Int64("profile_id", profileID), zap.Error(err))
		}
	}
}
