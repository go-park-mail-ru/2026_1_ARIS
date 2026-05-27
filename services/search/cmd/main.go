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
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/elasticsearch"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	searchHTTP "github.com/go-park-mail-ru/2026_1_ARIS/services/search/internal/handler/http"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/search/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load("services/search/.env", ".env")

	logg, err := logger.New("info")
	if err != nil {
		log.Fatal("fail to create logger: ", err)
	}
	defer func() {
		if err := logg.Sync(); err != nil {
			logg.Error("fail to sync logger", zap.Error(err))
		}
	}()

	esClient, err := elasticsearch.New()
	if err != nil {
		logg.Fatal("failed to create elasticsearch client", zap.Error(err))
	}

	mediaConn, err := grpc.NewClient(utils.EnvString("MEDIA_GRPC_ADDR", "localhost:8003"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logg.Fatal("failed to connect media grpc", zap.Error(err))
	}
	defer mediaConn.Close()

	searchUsecase := usecase.New(esClient, mediapb.NewMediaServiceClient(mediaConn))
	httpHandler := searchHTTP.New(searchUsecase, logg)

	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Route("/api", func(r chi.Router) {
		httpHandler.RegisterRoutes(r)
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("SEARCH_HTTP_PORT", 8088)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logg.Info("search http server started", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve search http", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info("search service is stopping")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		logg.Fatal("search http server forced to shutdown", zap.Error(err))
	}
	logg.Info("search service stopped")
}
