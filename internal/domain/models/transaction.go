package models

import (
	"github.com/shopspring/decimal"
	"time"
)

type Transaction struct {
	ID            string          `db:"id"`
	TransactionID string          `db:"transaction_id"`
	State         string          `db:"state"`
	Amount        decimal.Decimal `db:"amount"`
	SourceType    SourceType      `db:"source_id"`
	Processed     bool            `db:"processed"`
	User          User            `db:"user_id"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
}
