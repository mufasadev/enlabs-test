package interactor

import (
	"context"
	"fmt"
	"github.com/mufasadev/enlabs-test/internal/domain/repositories"
	"github.com/mufasadev/enlabs-test/internal/errors"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"github.com/rs/zerolog"
	"sync"
	"time"
)

type CancelTransactionInteractor struct {
	transactionRepository repositories.TransactionRepository
	logger                *zerolog.Logger
	sync.Mutex
	counter int
}

// NewCancelTransactionInteractor creates a new CancelTransactionInteractor
func NewCancelTransactionInteractor(transactionRepository repositories.TransactionRepository) *CancelTransactionInteractor {
	l := log.GetLogger()
	return &CancelTransactionInteractor{
		transactionRepository: transactionRepository,
		logger:                &l,
	}
}

// Execute will cancel all odd transactions and update the balance
func (c *CancelTransactionInteractor) Execute(ctx context.Context) error {
	c.Lock() // needed to test and print the counter. Should be removed in production
	defer c.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ids, err := c.transactionRepository.CancelOddTransactionsAndUpdateBalance(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg(errors.ErrFailedCancelOddTransactions)
		return err
	}

	for _, id := range ids {
		c.logger.Info().Msgf("Transaction canceled: %s", id)
	}

	c.counter++
	fmt.Println("Transactions canceled: ", len(ids), " | Total iterations: ", c.counter)

	return nil
}
