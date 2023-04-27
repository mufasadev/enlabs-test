package middlewares

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/errors"
	"github.com/mufasadev/enlabs-test/internal/usecases/interactor"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"net/http"
	"time"
)

// SourceTypeValidationMiddleware validates the source type header.
func SourceTypeValidationMiddleware(sourceTypeInt *interactor.SourceTypeInteractor) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			logger := log.GetLogger()
			sourceType := r.Header.Get("Source-Type")

			if sourceType == "" {
				logger.Error().Msg(errors.ErrSourceTypeRequired)
				errors.HandleHTTPError(w, errors.NewBadRequestError(errors.ErrSourceTypeRequired))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// TODO: add caching
			if exists, _ := sourceTypeInt.ExistsByName(ctx, sourceType); !exists {
				logger.Error().Msg(errors.ErrInvalidSourceType)
				errors.HandleHTTPError(w, errors.NewBadRequestError(errors.ErrInvalidSourceType))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
