package handlers

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/mufasadev/enlabs-test/internal/errors"
	http2 "github.com/mufasadev/enlabs-test/internal/infrastructure/api/http"
	"github.com/mufasadev/enlabs-test/internal/usecases/interactor"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"github.com/rs/zerolog"
	"net/http"
	"time"
)

type BalanceHandler struct {
	interactor *interactor.UserInteractor
	logger     *zerolog.Logger
}

func NewBalanceHandler(interactor *interactor.UserInteractor) *BalanceHandler {
	logger := log.GetLogger()
	return &BalanceHandler{interactor: interactor, logger: &logger}
}

func (uh *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, http2.UserIDParam)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	balance, err := uh.interactor.GetBalance(ctx, userId)
	if err != nil {
		uh.logger.Error().Err(err).Msg("failed to get balance")
		errors.HandleHTTPError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Balance float64 `json:"balance"`
	}{Balance: balance})
}
