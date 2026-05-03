package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	mymiddleware "github.com/go-park-mail-ru/2026_1_ARIS/internal/middleware"
	searchHTTP "github.com/go-park-mail-ru/2026_1_ARIS/internal/search/handler/http"
	searchRepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/search/repository"
	searchService "github.com/go-park-mail-ru/2026_1_ARIS/internal/search/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/config"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
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
	defer func() { _ = logg.Sync() }()

	envConf, err := config.NewConfig()
	if err != nil {
		logg.Fatal("fail to load env variables", zap.Error(err))
	}
	db, err := postgres.New(ctx, envConf)
	if err != nil {
		logg.Fatal("fail to connect PostgreSQL", zap.Error(err))
	}

	mediaConn, err := grpc.NewClient(envString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	store := searchRepo.NewStore(searchRepo.NewSearchStorage(db))
	searchSvc := searchService.New(store, mediapb.NewMediaServiceClient(mediaConn))
	httpHandler := searchHTTP.New(searchSvc)

	router := chi.NewRouter()
	router.Use(mymiddleware.RequestIDMiddleware(logg))
	router.Use(mymiddleware.AccessLogMiddleware(logg))
	router.Use(middleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r)
	})

	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", envInt("SEARCH_HTTP_PORT", 8088)), Handler: router}
	go func() {
		logg.Info("search http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve search http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("search http server forced to shutdown", zap.Error(err))
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
