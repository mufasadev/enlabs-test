package errors

import (
	"encoding/json"
	"net/http"
)

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HandleHTTPError handles http errors
func HandleHTTPError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	switch e := err.(type) {
	case *BadRequestError:
		httpErr = &HTTPError{
			Code:    http.StatusBadRequest,
			Message: e.Error(),
		}
	case *InsufficientFundsError:
		httpErr = &HTTPError{
			Code:    http.StatusBadRequest,
			Message: e.Error(),
		}
	case *TransactionDuplicateError:
		httpErr = &HTTPError{
			Code:    http.StatusUnprocessableEntity,
			Message: e.Error(),
		}
	default:
		httpErr = &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpErr.Code)
	json.NewEncoder(w).Encode(httpErr)
}
