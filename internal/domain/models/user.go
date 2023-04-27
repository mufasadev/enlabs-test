package models

import "github.com/shopspring/decimal"

type User struct {
	ID      string          `json:"id"`
	Balance decimal.Decimal `json:"balance"`
	Account string          `json:"account_number"`
}
