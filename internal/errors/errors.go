package errors

import (
	"errors"
	"fmt"
)

const (
	ErrFailedCancelOddTransactions    = "Failed to cancel odd transactions"
	ErrorFailedToConnectToTheDatabase = "Failed to connect to the database"
	ErrorFailedToRunTheServer         = "Failed to run the server"
	ErrorFailedToShutdownTheServer    = "Failed to shutdown the server"
	ErrFailedDecodeRequestBody        = "Failed to decode request body"
	ErrInvalidRequestBody             = "Invalid request body"
	ErrFailedProcessTransaction       = "Failed to process transaction"
	ErrSourceTypeRequired             = "Source-Type is required"
	ErrInvalidSourceType              = "Invalid Source-Type"
	ErrUserIDRequired                 = "User ID is required"
	ErrInvalidUserID                  = "Invalid User ID"
)

type BadRequestError struct {
	Message string
}

func NewBadRequestError(message string) *BadRequestError {
	return &BadRequestError{Message: message}
}

func (e *BadRequestError) Error() string {
	return fmt.Sprintf("Bad request: %s", e.Message)
}

type InsufficientFundsError struct{}

func NewInsufficientFundsError() *InsufficientFundsError {
	return &InsufficientFundsError{}
}

func (e *InsufficientFundsError) Error() string {
	return "insufficient funds"
}

type TransactionDuplicateError struct{}

func NewTransactionDuplicateError() *TransactionDuplicateError {
	return &TransactionDuplicateError{}
}

func (e *TransactionDuplicateError) Error() string {
	return "transaction already exists"
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
