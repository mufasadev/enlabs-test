package dtos

import "encoding/json"

type TransactionDTO struct {
	State         string          `json:"state"`
	Amount        string          `json:"-"`
	RawAmount     json.RawMessage `json:"amount"`
	TransactionID string          `json:"transactionId"`
}
