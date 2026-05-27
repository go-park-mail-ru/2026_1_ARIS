package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/elasticsearch"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/postgres"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/indexer/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/indexer/internal/worker"
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
}
