package handlers

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/mufasadev/enlabs-test/internal/errors"
	http2 "github.com/mufasadev/enlabs-test/internal/infrastructure/api/http"
	"github.com/mufasadev/enlabs-test/internal/usecases/dtos"
	"github.com/mufasadev/enlabs-test/internal/usecases/interactor"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"github.com/rs/zerolog"
	"net/http"
)

type TransactionHandler struct {
	interactor *interactor.TransactionInteractor
	logger     *zerolog.Logger
}

func NewTransactionHandler(interactor *interactor.TransactionInteractor) *TransactionHandler {
	logger := log.GetLogger()
	return &TransactionHandler{interactor: interactor, logger: &logger}
}

func (h *TransactionHandler) ProcessTransaction(w http.ResponseWriter, r *http.Request) {
	var dto dtos.TransactionDTO
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		h.logger.Error().Err(err).Msg(errors.ErrFailedDecodeRequestBody)
		errors.HandleHTTPError(w, errors.NewBadRequestError(errors.ErrInvalidRequestBody))
		return
	}
	var amount interface{}
	err = json.Unmarshal(dto.RawAmount, &amount)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to unmarshal raw amount")
		errors.HandleHTTPError(w, errors.NewBadRequestError("invalid amount"))
		return
	}
	dto.Amount = amount.(string)
	sourceType := r.Header.Get("Source-Type")
	userId := chi.URLParam(r, http2.UserIDParam)
	transaction, err := h.interactor.ProcessTransaction(userId, sourceType, &dto)
	if err != nil {
		h.logger.Error().Err(err).Msg(errors.ErrFailedProcessTransaction)
		errors.HandleHTTPError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transaction)
}
