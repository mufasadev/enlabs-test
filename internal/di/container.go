package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mufasadev/enlabs-test/internal/infrastructure/api/handlers"
	"github.com/mufasadev/enlabs-test/internal/infrastructure/database/repositories"
	"github.com/mufasadev/enlabs-test/internal/usecases/interactor"
)

type Container struct {
	TransactionHandler          *handlers.TransactionHandler
	SourceTypeInteractor        *interactor.SourceTypeInteractor
	UserInteractor              *interactor.UserInteractor
	CancelTransactionInteractor *interactor.CancelTransactionInteractor
	BalanceHandler              *handlers.BalanceHandler
}

// NewContainer creates a new Container instance.
func NewContainer(db *pgxpool.Pool) *Container {
	transactionRepository := repositories.NewTransactionRepositoryImpl(db)
	userRepository := repositories.NewUserRepositoryImpl(db)
	sourceTypeRepository := repositories.NewSourceTypeRepositoryImpl(db)

	transactionInteractor := interactor.NewTransactionInteractor(transactionRepository, userRepository, sourceTypeRepository)
	transactionHandler := handlers.NewTransactionHandler(transactionInteractor)

	sourceTypeInteractor := interactor.NewSourceTypeInteractor(sourceTypeRepository)

	userInteractor := interactor.NewUserInteractor(userRepository)

	cancelTransactionInteractor := interactor.NewCancelTransactionInteractor(transactionRepository)

	balanceInteractor := interactor.NewUserInteractor(userRepository)
	balanceHandler := handlers.NewBalanceHandler(balanceInteractor)

	return &Container{
		TransactionHandler:          transactionHandler,
		SourceTypeInteractor:        sourceTypeInteractor,
		UserInteractor:              userInteractor,
		CancelTransactionInteractor: cancelTransactionInteractor,
		BalanceHandler:              balanceHandler,
	}
}
