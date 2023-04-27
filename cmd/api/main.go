package main

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/app"
	"github.com/mufasadev/enlabs-test/internal/config"
	"github.com/mufasadev/enlabs-test/internal/di"
	"github.com/mufasadev/enlabs-test/internal/errors"
	"github.com/mufasadev/enlabs-test/internal/infrastructure/api/routers"
	"github.com/mufasadev/enlabs-test/internal/infrastructure/database/db_client"
	"github.com/mufasadev/enlabs-test/pkg/log"
)

const (
	appName = "enlabs-test"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	log.Init(appName, log.WithConsoleLogger())
	logger := log.GetLogger()

	pgClient := db_client.NewPGClient(cfg.PostgreSQL)
	db, err := pgClient.Connect()
	if err != nil {
		logger.Fatal().Err(err).Msg(errors.ErrorFailedToConnectToTheDatabase)
	}

	container := di.NewContainer(db)

	cancelTx := app.NewCancelTransactionProcess(container.CancelTransactionInteractor, cfg.Process)
	go cancelTx.Run(ctx)

	router := routers.NewRouter(container)
	service := app.NewService(cfg)
	service.Run(ctx, router)
}
