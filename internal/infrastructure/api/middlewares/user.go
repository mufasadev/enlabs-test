package middlewares

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/mufasadev/enlabs-test/internal/errors"
	http2 "github.com/mufasadev/enlabs-test/internal/infrastructure/api/http"
	"github.com/mufasadev/enlabs-test/internal/usecases/interactor"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"net/http"
	"time"
)

// UserValidationMiddleware validates the user id.
func UserValidationMiddleware(userInt *interactor.UserInteractor) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.GetLogger()
			userId := chi.URLParam(r, http2.UserIDParam)
			if userId == "" { // test id "f60ae2e1-ee72-4a6a-bef2-7cde5c83782f"
				logger.Error().Msg(errors.ErrUserIDRequired)
				errors.HandleHTTPError(w, errors.NewBadRequestError(errors.ErrUserIDRequired))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if exists, _ := userInt.ExistsByID(ctx, userId); !exists {
				logger.Error().Msg(errors.ErrInvalidUserID)
				errors.HandleHTTPError(w, errors.NewBadRequestError(errors.ErrInvalidUserID))
				return
			}

			rctx := chi.RouteContext(r.Context())
			rctx.URLParams.Add(http2.UserIDParam, userId)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			next.ServeHTTP(w, r)
		})
	}
}
