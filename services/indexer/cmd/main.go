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
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/elasticsearch"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/indexer/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/indexer/internal/worker"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load("services/indexer/.env", ".env")

	logg, err := logger.New("info")
	if err != nil {
		log.Fatal("fail to create logger: ", err)
	}
	defer func() {
		if err := logg.Sync(); err != nil {
			logg.Error("fail to sync logger", zap.Error(err))
		}
	}()

	ctx := logger.WithLogger(context.Background(), logg)
	metricsServer := startMetricsServer(logg)

	db, err := postgres.New(ctx)
	if err != nil {
		logg.Fatal("fail to connect PostgreSQL", zap.Error(err))
	}

	esClient, err := elasticsearch.New()
	if err != nil {
		logg.Fatal("fail to create elasticsearch client", zap.Error(err))
	}

	if err := elasticsearch.EnsureIndices(ctx, esClient); err != nil {
		logg.Fatal("fail to ensure elasticsearch indices", zap.Error(err))
	}

	outboxRepo := repository.NewOutboxRepo(db)
	fetcherRepo := repository.NewFetcherRepo(db)
	esWriter := repository.NewESWriter(esClient)

	locked, err := outboxRepo.TryAdvisoryLock(ctx)
	if err != nil {
		logg.Fatal("advisory lock check failed", zap.Error(err))
	}
	if !locked {
		logg.Fatal("another indexer instance is already running (advisory lock held)")
		os.Exit(1)
	}
	logg.Info("acquired advisory lock, starting indexer")

	seeded, err := outboxRepo.SeedIfFirstRun(ctx)
	if err != nil {
		logg.Fatal("failed to seed outbox on first run", zap.Error(err))
	}
	if seeded > 0 {
		logg.Info("seeded search_outbox with existing entities", zap.Int64("count", seeded))
	}

	w := worker.New(outboxRepo, fetcherRepo, esWriter)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go w.Run(ctx)

	<-stop
	logg.Info("indexer stopping")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logg.Warn("indexer metrics server forced to shutdown", zap.Error(err))
	}
}

func startMetricsServer(logg *zap.Logger) *http.Server {
	router := chi.NewRouter()
	metrics.RegisterHTTP(router, "indexer")
	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", utils.EnvInt("INDEXER_METRICS_PORT", 8090)),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logg.Info("indexer metrics server started", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal("failed to serve indexer metrics", zap.Error(err))
		}
	}()

	return server
}
