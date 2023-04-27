package repositories

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mufasadev/enlabs-test/internal/config"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
	apperr "github.com/mufasadev/enlabs-test/internal/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var (
	userId       = "f60ae2e1-ee72-4a6a-bef2-7cde5c83782f"
	sourceTypeId = "8bd85576-8d8c-47b8-bfa8-6e9d2fc4d267"
	db           *pgxpool.Pool
)

func randDecimal(add int) decimal.Decimal {
	rand.Seed(time.Now().UnixNano())
	min := 0    // min amount
	max := 99   // max amount
	mind := 111 // min decimal
	maxd := 999 // max decimal
	num := rand.Intn(max-min) + min
	numd := rand.Intn(maxd-mind) + mind
	d, err := decimal.NewFromString(fmt.Sprintf("%d.%d", num+add, numd))
	if err != nil {
		return decimal.Decimal{}
	}
	return d
}

func TestInsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(t *testing.T) {
	setupDB()
	defer db.Close()

	err := truncateTransactionsTable(db)
	require.NoError(t, err)

	transactionRepo := NewTransactionRepositoryImpl(db)

	t.Run("successful transaction", func(t *testing.T) {
		transaction := &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "win",
			Amount:        randDecimal(0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(context.Background(), transaction)

		assert.NoError(t, err)
	})

	t.Run("transaction leading to negative balance", func(t *testing.T) {
		transaction := &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "lost",
			Amount:        decimal.NewFromInt(1000000),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(context.Background(), transaction)
		assert.True(t, errors.Is(err, apperr.NewInsufficientFundsError()))
	})

	t.Run("balance_non-negative", func(t *testing.T) {
		err = truncateTransactionsTable(db)
		require.NoError(t, err)

		n := 1000 // transactions count
		results := make(chan error, n*2)
		var wg sync.WaitGroup
		wg.Add(2)

		// add balance
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         "win",
					Amount:        randDecimal(0),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}
				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(context.Background(), transaction)
				results <- err
			}
		}()

		// subtract balance
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         "lost",
					Amount:        randDecimal(100),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}
				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(context.Background(), transaction)
				results <- err
			}
		}()

		wg.Wait()
		close(results)

		var errorCount int
		for err = range results {
			if err != nil {
				errorCount++
				assert.True(t, errors.Is(err, apperr.NewInsufficientFundsError()))
			}
		}

		assert.True(t, errorCount < n*2, "Too many transactions lead to a negative balance")

		// check final balance is non-negative
		var balance float64
		err = db.QueryRow(context.Background(), fmt.Sprintf("SELECT balance FROM users WHERE id = '%s'", userId)).Scan(&balance)
		require.NoError(t, err)
		assert.True(t, balance >= 0, "The final balance must be non-negative")
	})

	t.Run("concurrent", func(t *testing.T) {
		err = truncateTransactionsTable(db)
		require.NoError(t, err)
		expectedBalance := 1000.0
		err = setInitialUserBalance(db, expectedBalance)
		require.NoError(t, err)

		numTransactions := 1000
		amounts := make([]float64, numTransactions)

		for i := 0; i < numTransactions; i++ {
			amounts[i] = float64(i%10 + 1)
			if i%2 == 0 {
				amounts[i] = -amounts[i]
			}
		}

		var wg sync.WaitGroup
		wg.Add(numTransactions)

		for i := 0; i < numTransactions; i++ {
			go func(i int) {
				defer wg.Done()

				state := "win"
				if amounts[i] < 0 {
					state = "lost"
				}

				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         state,
					Amount:        decimal.NewFromFloat(math.Abs(amounts[i])),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}

				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(context.Background(), transaction)
				if err != nil && !errors.Is(err, apperr.NewInsufficientFundsError()) {
					t.Error(err)
				}
			}(i)
		}

		wg.Wait()

		// check final balance
		var finalBalance float64
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&finalBalance)
		require.NoError(t, err)

		for _, amount := range amounts {
			if expectedBalance+amount >= 0 {
				expectedBalance += amount
			}
		}

		assert.Equal(t, expectedBalance, finalBalance, "The final balance must be equal to the expected balance")
	})

	t.Run("concurrent_with_reads", func(t *testing.T) {
		err = truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 1000.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		numTransactions := 1000
		amounts := make([]float64, numTransactions)

		for i := 0; i < numTransactions; i++ {
			amounts[i] = float64(i%10 + 1)
			if i%2 == 0 {
				amounts[i] = -amounts[i]
			}
		}

		var wg, wgRead sync.WaitGroup
		wg.Add(numTransactions)
		wgRead.Add(numTransactions)

		readData := func() {
			defer wgRead.Done()

			var balance float64
			err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&balance)
			if err != nil {
				t.Error(err)
			}
		}

		for i := 0; i < numTransactions; i++ {
			go readData()
			go func(i int) {
				defer wg.Done()

				state := "win"
				if amounts[i] < 0 {
					state = "lost"
				}

				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         state,
					Amount:        decimal.NewFromFloat(math.Abs(amounts[i])),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}

				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(context.Background(), transaction)
				if err != nil && !errors.Is(err, apperr.NewInsufficientFundsError()) {
					t.Error(err)
				}
			}(i)
		}

		wg.Wait()
		wgRead.Wait()

		// check final balance
		var finalBalance float64
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&finalBalance)
		require.NoError(t, err)

		expectedBalance := initialBalance
		for _, amount := range amounts {
			if expectedBalance+amount >= 0 {
				expectedBalance += amount
			}
		}

		assert.Equal(t, expectedBalance, finalBalance, "The final balance must be equal to the expected balance")
	})
}

func TestInsertTransactionAndUpdateUserBalanceWithCreatingTransaction(t *testing.T) {
	setupDB()
	defer db.Close()

	err := truncateTransactionsTable(db)
	require.NoError(t, err)

	transactionRepo := NewTransactionRepositoryImpl(db)

	t.Run("successful transaction", func(t *testing.T) {
		transaction := &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "win",
			Amount:        randDecimal(0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)

		assert.NoError(t, err)
	})

	t.Run("transaction leading to negative balance", func(t *testing.T) {
		transaction := &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "lost",
			Amount:        decimal.NewFromInt(1000000),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
		assert.True(t, errors.Is(err, apperr.NewInsufficientFundsError()))
	})

	t.Run("balance_non-negative", func(t *testing.T) {
		err = truncateTransactionsTable(db)
		require.NoError(t, err)

		n := 1000 // transactions count
		results := make(chan error, n*2)
		var wg sync.WaitGroup
		wg.Add(2)

		// add balance
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         "win",
					Amount:        randDecimal(0),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}
				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
				results <- err
			}
		}()

		// subtract balance
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         "lost",
					Amount:        randDecimal(100),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}
				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
				results <- err
			}
		}()

		wg.Wait()
		close(results)

		var errorCount int
		for err = range results {
			if err != nil {
				errorCount++
				assert.True(t, errors.Is(err, apperr.NewInsufficientFundsError()))
			}
		}

		assert.True(t, errorCount < n*2, "Too many transactions lead to a negative balance")

		// check final balance is non-negative
		var balance float64
		err = db.QueryRow(context.Background(), fmt.Sprintf("SELECT balance FROM users WHERE id = '%s'", userId)).Scan(&balance)
		require.NoError(t, err)
		assert.True(t, balance >= 0, "The final balance must be non-negative")
	})

	t.Run("concurrent", func(t *testing.T) {
		err = truncateTransactionsTable(db)
		require.NoError(t, err)
		expectedBalance := 1000.0
		err = setInitialUserBalance(db, expectedBalance)
		require.NoError(t, err)

		numTransactions := 1000
		amounts := make([]float64, numTransactions)

		for i := 0; i < numTransactions; i++ {
			amounts[i] = float64(i%10 + 1)
			if i%2 == 0 {
				amounts[i] = -amounts[i]
			}
		}

		var wg sync.WaitGroup
		wg.Add(numTransactions)

		for i := 0; i < numTransactions; i++ {
			go func(i int) {
				defer wg.Done()

				state := "win"
				if amounts[i] < 0 {
					state = "lost"
				}

				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         state,
					Amount:        decimal.NewFromFloat(math.Abs(amounts[i])),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}

				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
				if err != nil && !errors.Is(err, apperr.NewInsufficientFundsError()) {
					t.Error(err)
				}
			}(i)
		}

		wg.Wait()

		// check final balance
		var finalBalance float64
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&finalBalance)
		require.NoError(t, err)

		for _, amount := range amounts {
			if expectedBalance+amount >= 0 {
				expectedBalance += amount
			}
		}

		assert.Equal(t, expectedBalance, finalBalance, "The final balance must be equal to the expected balance")
	})

	t.Run("concurrent_with_reads", func(t *testing.T) {
		err = truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 1000.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		numTransactions := 1000
		amounts := make([]float64, numTransactions)

		for i := 0; i < numTransactions; i++ {
			amounts[i] = float64(i%10 + 1)
			if i%2 == 0 {
				amounts[i] = -amounts[i]
			}
		}

		var wg, wgRead sync.WaitGroup
		wg.Add(numTransactions)
		wgRead.Add(numTransactions)

		readData := func() {
			defer wgRead.Done()

			var balance float64
			err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&balance)
			if err != nil {
				t.Error(err)
			}
		}

		for i := 0; i < numTransactions; i++ {
			go readData()
			go func(i int) {
				defer wg.Done()

				state := "win"
				if amounts[i] < 0 {
					state = "lost"
				}

				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         state,
					Amount:        decimal.NewFromFloat(math.Abs(amounts[i])),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}

				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
				if err != nil && !errors.Is(err, apperr.NewInsufficientFundsError()) {
					t.Error(err)
				}
			}(i)
		}

		wg.Wait()
		wgRead.Wait()

		// check final balance
		var finalBalance float64
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&finalBalance)
		require.NoError(t, err)

		expectedBalance := initialBalance
		for _, amount := range amounts {
			if expectedBalance+amount >= 0 {
				expectedBalance += amount
			}
		}

		assert.Equal(t, expectedBalance, finalBalance, "The final balance must be equal to the expected balance")
	})
}

func TestCancelOddTransactionsAndUpdateBalance(t *testing.T) {
	setupDB()
	defer db.Close()

	transactionRepo := NewTransactionRepositoryImpl(db)

	t.Run("successful_transaction", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 1000.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)
		numTransactions := 1000
		amounts := make([]float64, numTransactions)

		for i := 0; i < numTransactions; i++ {
			amounts[i] = float64(i%10 + 1)
			if i%2 == 0 {
				amounts[i] = -amounts[i] + 0.5
			}
		}
		for i := 0; i < numTransactions; i++ {
			func(i int) {
				state := "win"
				if amounts[i] < 0 {
					state = "lost"
				}

				transaction := &models.Transaction{
					TransactionID: uuid.New().String(),
					State:         state,
					Amount:        decimal.NewFromFloat(math.Abs(amounts[i])),
					SourceType:    models.SourceType{ID: sourceTypeId},
					User:          models.User{ID: userId},
				}

				_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
				if err != nil {
					t.Error(err)
				}
			}(i)
		}

		ids, err := transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())

		assert.Nil(t, err)
		assert.Equal(t, 10, len(ids))

	})

	t.Run("concurrent_calls", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 1000.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		n := 1000
		errCh := make(chan error, n)
		for i := 0; i < n; i++ {
			go func() {
				_, err = transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())
				errCh <- err
			}()
		}

		// Collect results
		successCount := 0
		for i := 0; i < n; i++ {
			err = <-errCh
			if err == nil {
				successCount++
			} else {
				fmt.Println("Error:", err)
			}
		}

		assert.True(t, successCount > 0, "At least one call should succeed")
	})

	t.Run("less_than_10", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 0.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		n := 2
		for i := 0; i < n; i++ {
			state := "win"

			transaction := &models.Transaction{
				TransactionID: uuid.New().String(),
				State:         state,
				Amount:        randDecimal(0),
				SourceType:    models.SourceType{ID: sourceTypeId},
				User:          models.User{ID: userId},
			}

			_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
			if err != nil {
				t.Error(err)
			}
		}

		r, err := transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())
		if err != nil {
			t.Error(err)
		}
		assert.True(t, len(r) == 1, "Only one transaction should be canceled")
	})

	// Test if no transactions are canceled
	t.Run("no_transactions", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 0.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		r, err := transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())
		if err != nil {
			t.Error(err)
		}
		assert.True(t, len(r) == 0, "No transactions should be canceled")
	})

	// Test order of transactions
	t.Run("odd_rows_by_created_order", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 0.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		transaction := &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "win",
			Amount:        decimal.NewFromFloat(1.0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
		if err != nil {
			t.Error(err)
		}

		transaction = &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "win",
			Amount:        decimal.NewFromFloat(100.0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
		if err != nil {
			t.Error(err)
		}

		r, err := transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())
		if err != nil {
			t.Error(err)
		}
		assert.True(t, len(r) == 1, "Only one transaction should be canceled")

		var finalBalance decimal.Decimal
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&finalBalance)
		require.NoError(t, err)

		assert.True(t, finalBalance.Equal(decimal.NewFromFloat(1.0)), "Balance should be 1.0")
	})

	// Test lost state transactions canceling
	t.Run("non_negative_value_lost", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 300.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		transaction := &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "lost",
			Amount:        decimal.NewFromFloat(100.0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}
		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
		if err != nil {
			t.Error(err)
		}

		transaction = &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "lost",
			Amount:        decimal.NewFromFloat(100.0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}
		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
		if err != nil {
			t.Error(err)
		}

		transaction = &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "lost",
			Amount:        decimal.NewFromFloat(1.0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
		if err != nil {
			t.Error(err)
		}

		transaction = &models.Transaction{
			TransactionID: uuid.New().String(),
			State:         "lost",
			Amount:        decimal.NewFromFloat(1.0),
			SourceType:    models.SourceType{ID: sourceTypeId},
			User:          models.User{ID: userId},
		}

		_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
		if err != nil {
			t.Error(err)
		}

		require.NoError(t, err)

		r, err := transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())
		if err != nil {
			t.Error(err)
		}

		assert.True(t, len(r) == 2, "Two transactions should be canceled")

		var finalBalance decimal.Decimal
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&finalBalance)
		require.NoError(t, err)

		assert.True(t, finalBalance.Equal(decimal.NewFromFloat(199.0)), "Balance should be 199.0")
	})

	// Test win state transactions canceling
	t.Run("non_negative_value_win", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 0.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		for i := 0; i < 6; i++ {
			transaction := &models.Transaction{
				TransactionID: uuid.New().String(),
				State:         "win",
				Amount:        decimal.NewFromFloat(50.0),
				SourceType:    models.SourceType{ID: sourceTypeId},
				User:          models.User{ID: userId},
			}

			_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
			if err != nil {
				t.Error(err)
			}
		}

		initialBalance = 100.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		r, err := transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())
		if err != nil {
			t.Error(err)
		}

		assert.True(t, len(r) == 2, "Only 2 of 3 transaction should be canceled")

		var finalBalance decimal.Decimal
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&finalBalance)
		require.NoError(t, err)

		assert.True(t, finalBalance.Equal(decimal.NewFromFloat(0.0)), "Balance should be 0.0")
	})

	// Test for concurrent calls of CancelOddTransactionsAndUpdateBalance method and InsertTransactionAndUpdateUserBalanceWithCreatingTransaction
	t.Run("concurrent_calls_cross_func", func(t *testing.T) {
		err := truncateTransactionsTable(db)
		require.NoError(t, err)
		initialBalance := 1000.0
		err = setInitialUserBalance(db, initialBalance)
		require.NoError(t, err)

		n := 100
		iterations := 10
		errCh := make(chan error, n*iterations*2)
		wg := sync.WaitGroup{}
		wg.Add(n)

		mu := sync.RWMutex{}

		expectedBalanceAdded := 0.0
		expectedBalanceCanceled := 0.0
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					state := "win"
					amount := float64(j%10 + 1)
					if j%2 == 0 {
						state = "lost"
						amount = -amount + 0.5
					}

					transaction := &models.Transaction{
						TransactionID: uuid.New().String(),
						State:         state,
						Amount:        decimal.NewFromFloat(math.Abs(amount)),
						SourceType:    models.SourceType{ID: sourceTypeId},
						User:          models.User{ID: userId},
					}

					_, err = transactionRepo.InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(context.Background(), transaction)
					if err == nil {
						mu.Lock()
						expectedBalanceAdded += amount
						mu.Unlock()
					}
					errCh <- err

					rows, err := transactionRepo.CancelOddTransactionsAndUpdateBalance(context.Background())
					if err == nil {
						mu.Lock()
						for _, row := range rows {
							f, _ := row.Amount.Float64()
							if row.State == "win" {
								expectedBalanceCanceled -= f
							} else {
								expectedBalanceCanceled += f
							}
						}
						mu.Unlock()
					}
					errCh <- err
				}
			}()
		}

		wg.Wait()
		close(errCh)

		successCount := 0
		for err = range errCh {
			if err == nil {
				successCount++
			} else {
				fmt.Println("Error:", err)
			}
		}

		assert.True(t, successCount > 0, "At least one call should succeed")

		var actualBalance decimal.Decimal
		err = db.QueryRow(context.Background(), "SELECT balance FROM users WHERE id = $1", userId).Scan(&actualBalance)
		require.NoError(t, err)

		ib := decimal.NewFromFloat(initialBalance).Round(2)

		assert.True(t, actualBalance.Equal(ib), "Balance should be equal to initial balance")
		assert.True(t, expectedBalanceAdded+expectedBalanceCanceled == 0.0, "Sum of added and canceled transactions should be 0.0")
	})
}

// Test helpers and setup functions
// =================================
// Setup DB
func setupDB() {
	cnf := config.Load()

	config, err := pgxpool.ParseConfig(cnf.DSN())
	if err != nil {
		panic(err)
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	db, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}
}

// Truncate transactions table
func truncateTransactionsTable(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), "TRUNCATE TABLE transactions")
	return err
}

// Set initial user balance
func setInitialUserBalance(db *pgxpool.Pool, balance float64) error {
	_, err := db.Exec(context.Background(), "UPDATE users SET balance = $1 WHERE id = $2", balance, userId)
	return err
}
