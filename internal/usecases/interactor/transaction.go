package interactor

import (
	"context"
	"fmt"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
	"github.com/mufasadev/enlabs-test/internal/domain/repositories"
	apperrors "github.com/mufasadev/enlabs-test/internal/errors"
	"github.com/mufasadev/enlabs-test/internal/usecases/dtos"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"time"
)

type TransactionInteractor struct {
	transactionRepository repositories.TransactionRepository
	userRepository        repositories.UserRepository
	sourceType            repositories.SourceTypeRepository
	logger                *zerolog.Logger
}

func NewTransactionInteractor(transactionRepository repositories.TransactionRepository, userRepository repositories.UserRepository, sourceType repositories.SourceTypeRepository) *TransactionInteractor {
	l := log.GetLogger()
	return &TransactionInteractor{
		transactionRepository: transactionRepository,
		userRepository:        userRepository,
		sourceType:            sourceType,
		logger:                &l,
	}
}

func (i *TransactionInteractor) ProcessTransaction(userID string, sourceType string, dto *dtos.TransactionDTO) (*repositories.TransactionRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := i.userRepository.GetByID(ctx, userID)
	if err != nil {
		i.logger.Error().Err(err).Msg("Failed to get user")
		return nil, apperrors.NewBadRequestError("Invalid user ID")
	}

	// check if transaction exists TODO: add caching
	tx, err := i.transactionRepository.GetByTransactionID(ctx, dto.TransactionID)
	if err != nil {
		return nil, err
	}

	if tx != nil {
		return nil, apperrors.NewTransactionDuplicateError()
	}

	// check if source type exists
	source, err := i.sourceType.GetByName(ctx, sourceType)
	if err != nil {
		i.logger.Error().Err(err).Msg("Failed to get source type")
		return nil, apperrors.NewBadRequestError("Invalid source type")
	}

	if _, ok := models.ValidStates[dto.State]; !ok {
		i.logger.Error().Err(err).Msg("Invalid state")
		return nil, apperrors.NewBadRequestError("Invalid state")
	}

	if source == nil {
		return nil, apperrors.NewBadRequestError("Invalid source type")
	}

	var amount decimal.Decimal
	fmt.Println(dto.Amount)
	if amount, err = decimal.NewFromString(dto.Amount); err != nil {
		i.logger.Error().Err(err).Msg("Failed to parse amount")
		return nil, apperrors.NewBadRequestError("Invalid amount")
	}

	transaction := &models.Transaction{
		TransactionID: dto.TransactionID,
		State:         dto.State,
		Amount:        amount.Abs().Round(2),
		SourceType:    *source,
		User:          *user,
		Processed:     false,
	}

	data, err := i.transactionRepository.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(ctx, transaction)
	if err != nil {
		return nil, err
	}

	transaction.Processed = true
	return &data, nil
}
