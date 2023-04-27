package routers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mufasadev/enlabs-test/internal/di"
	http2 "github.com/mufasadev/enlabs-test/internal/infrastructure/api/http"
	"github.com/mufasadev/enlabs-test/internal/infrastructure/api/middlewares"
)

func NewRouter(container *di.Container) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	// Set up v1 routes with a path prefix
	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Route(fmt.Sprintf("/{%s}", http2.UserIDParam), func(r chi.Router) { // test id "f60ae2e1-ee72-4a6a-bef2-7cde5c83782f"
				r.Use(middlewares.UserValidationMiddleware(container.UserInteractor))
				r.Route("/transactions", func(r chi.Router) {
					r.Use(middlewares.SourceTypeValidationMiddleware(container.SourceTypeInteractor))
					th := container.TransactionHandler
					r.Post("/", th.ProcessTransaction)
				})
				r.Route("/balance", func(r chi.Router) {
					bh := container.BalanceHandler
					r.Get("/", bh.GetBalance)
				})
			})
		})
	})

	return router
}
