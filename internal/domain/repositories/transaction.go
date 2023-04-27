package repositories

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
	"github.com/shopspring/decimal"
)

const (
	SerializationError   = "40001"
	UniqueViolationError = "23505"
)

type TransactionRepository interface {
	GetByTransactionID(ctx context.Context, transactionID string) (*models.Transaction, error)
	InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(ctx context.Context, transaction *models.Transaction) (TransactionRow, error)
	GetUserBalance(ctx context.Context, userId string) (*decimal.Decimal, error)
	InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(ctx context.Context, transaction *models.Transaction) (TransactionRow, error)
	CancelOddTransactionsAndUpdateBalance(ctx context.Context) ([]CancelOddTransactionsAndUpdateBalanceRow, error)
}

type CancelOddTransactionsAndUpdateBalanceRow struct {
	UserId        string
	UserBalance   decimal.Decimal
	TransactionId string
	State         string
	Amount        decimal.Decimal
}

type TransactionRow struct {
	UserId        string
	UserBalance   float64
	TransactionId string
	Processed     bool
}
