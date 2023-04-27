package app

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/mufasadev/enlabs-test/internal/config"
	"github.com/mufasadev/enlabs-test/internal/errors"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Service struct {
	config *config.Config
	logger *zerolog.Logger
}

// NewService creates a new instance of the service
func NewService(cfg *config.Config) *Service {
	l := log.GetLogger()
	return &Service{config: cfg, logger: &l}
}

// Run starts the server and listens for incoming requests
func (s *Service) Run(ctx context.Context, router chi.Router) {
	server := &http.Server{
		Addr:    s.config.Server.Addr(),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal().Err(err).Msg(errors.ErrorFailedToRunTheServer)
		}
	}()

	s.logger.Info().Msg(fmt.Sprintf("Server is listening on %s", s.config.Server.Addr()))
	done := make(chan struct{})
	go s.shutdown(ctx, server, done)
	<-done
}

// shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Service) shutdown(ctx context.Context, server *http.Server, done chan struct{}) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		s.logger.Info().Msg("Server is shutting down due to context cancellation...")
	case <-quit:
		s.logger.Info().Msg("Server is shutting down...")
	}

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		s.logger.Error().Err(err).Msg(errors.ErrorFailedToShutdownTheServer)
	}

	s.logger.Info().Msg("Server stopped")
	close(done)
}
