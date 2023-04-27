package repositories

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
	"github.com/mufasadev/enlabs-test/internal/domain/repositories"
	apperrors "github.com/mufasadev/enlabs-test/internal/errors"
	"github.com/mufasadev/enlabs-test/pkg/log"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

type TransactionRepositoryImpl struct {
	db     *pgxpool.Pool
	logger *zerolog.Logger
}

// NewTransactionRepositoryImpl creates new instance of TransactionRepositoryImpl.
func NewTransactionRepositoryImpl(db *pgxpool.Pool) repositories.TransactionRepository {
	l := log.GetLogger()
	return &TransactionRepositoryImpl{
		db:     db,
		logger: &l,
	}
}

const withoutCreatingTransaction = `
WITH new_transaction AS (
  INSERT INTO transactions (transaction_id, state, amount, source_id, user_id, processed)
  VALUES ($1, $2, $3::NUMERIC(10,2), $4, $5, $6)
  RETURNING transaction_id, state, amount, user_id, processed
),
amount_change AS (
  SELECT (CASE
            WHEN state = 'win' THEN amount
            WHEN state = 'lost' THEN -amount
          END) AS change
  FROM new_transaction
),
updated_balance AS (
  UPDATE users
  SET balance = balance + (SELECT change FROM amount_change)
  WHERE id = (SELECT user_id FROM new_transaction) AND balance + (SELECT change FROM amount_change) >= 0
  RETURNING id, balance
),
final_result AS (
  SELECT updated_balance.id AS user_id, updated_balance.balance AS user_balance, new_transaction.transaction_id, new_transaction.processed
  FROM updated_balance, new_transaction
)
SELECT user_id, user_balance, transaction_id, processed FROM final_result;`

// InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction inserts transaction and updates user balance in a single transaction.
func (r *TransactionRepositoryImpl) InsertTransactionAndUpdateUserBalanceWithoutCreatingTransaction(ctx context.Context, transaction *models.Transaction) (repositories.TransactionRow, error) {

	args := []interface{}{
		transaction.TransactionID,
		transaction.State,
		transaction.Amount,
		transaction.SourceType.ID,
		transaction.User.ID,
		true,
	}

	var data repositories.TransactionRow
	var err error
	for {
		data, err = r.processTransactionWithQuery(ctx, withoutCreatingTransaction, true, args...)

		if err == nil {
			_ = data
			return data, nil
		}

		if isSerializationError(err) {
			// retry transaction if serialization error occurs (SQLSTATE 40001)
			continue
		} else {
			if errors.Is(err, pgx.ErrNoRows) {
				return data, apperrors.NewInsufficientFundsError()
			}
			return data, fmt.Errorf("transaction error: %w", err)
		}
	}
}

const withCreatingTransaction = `
WITH updated_balance AS (
  UPDATE users
  SET balance = balance + (CASE WHEN $2 = 'win' THEN $3::NUMERIC(10,2) WHEN $2 = 'lost' THEN $3::NUMERIC(10,2) * -1 END)
  WHERE id = $5 AND balance + (CASE WHEN $2 = 'win' THEN $3::NUMERIC(10,2) WHEN $2 = 'lost' THEN $3::NUMERIC(10,2) * -1 END) >= 0
  RETURNING id, balance
),
new_transaction AS (
  INSERT INTO transactions (transaction_id, state, amount, source_id, user_id, processed)
  VALUES ($1, $2, $3::NUMERIC(10,2), $4, $5, EXISTS (SELECT 1 FROM updated_balance))
  RETURNING transaction_id, processed
),
final_result AS (
  SELECT
    COALESCE((SELECT id FROM updated_balance), $5) AS uid,
    COALESCE((SELECT balance FROM updated_balance), (SELECT balance FROM users WHERE id = $5)) AS ub,
    new_transaction.transaction_id AS tid,
    new_transaction.processed AS p
  FROM new_transaction
)
SELECT uid, ub, tid, p FROM final_result;`

// InsertTransactionAndUpdateUserBalanceWithCreatingTransaction inserts transaction and updates user balance in a single transaction.
func (r *TransactionRepositoryImpl) InsertTransactionAndUpdateUserBalanceWithCreatingTransaction(ctx context.Context, transaction *models.Transaction) (repositories.TransactionRow, error) {
	args := []interface{}{
		transaction.TransactionID,
		transaction.State,
		transaction.Amount,
		transaction.SourceType.ID,
		transaction.User.ID,
	}

	var data repositories.TransactionRow
	var pgErr *pgconn.PgError
	for {
		tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if err != nil {
			return data, err
		}

		err = tx.QueryRow(ctx, withCreatingTransaction, args...).Scan(&data.UserId, &data.UserBalance, &data.TransactionId, &data.Processed)
		if err != nil {
			r.logger.Error().Err(err).Msg("transaction error")
			tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
			if err == nil {
				_ = data
				if data.Processed {
					return data, nil
				} else {
					return data, apperrors.NewInsufficientFundsError()
				}
			} else {
				r.logger.Error().Err(err).Msg("transaction error")
				tx.Rollback(ctx)
			}
		}

		if isSerializationError(err) {
			// retry transaction if serialization error occurs (SQLSTATE 40001)
			continue
		} else {
			if of := errors.As(err, &pgErr); of && pgErr.SQLState() == repositories.UniqueViolationError {
				return data, apperrors.NewTransactionDuplicateError()
			}
			return data, fmt.Errorf("transaction error: %w", err)
		}
	}
}

// processTransactionWithQuery processes transaction with given query.
func (r *TransactionRepositoryImpl) processTransactionWithQuery(ctx context.Context, query string, rollbackOnNoFunds bool, args ...interface{}) (repositories.TransactionRow, error) {
	var tr repositories.TransactionRow
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return tr, err
	}

	err = tx.QueryRow(ctx, query, args...).Scan(&tr.UserId, &tr.UserBalance, &tr.TransactionId, &tr.Processed)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) && !rollbackOnNoFunds {
			// Do nothing or add any specific action if needed
		} else {
			tx.Rollback(ctx)
			return tr, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		tx.Rollback(ctx)
		return tr, err
	}

	return tr, nil
}

const cancelOddTransactions = `
WITH odd_transactions_to_cancel AS (
  SELECT id, user_id, state, amount, created_at, transaction_id
  FROM (
    SELECT id, state, amount, user_id, transaction_id, created_at,
           DENSE_RANK() OVER (ORDER BY created_at DESC) AS rank
    FROM transactions
    WHERE processed = TRUE
    LIMIT 20
  ) ranked_transactions
  WHERE rank <= 20 AND rank % 2 = 1
),
processable_transactions AS (
  SELECT t.id, t.state, t.user_id, t.amount,
         SUM(CASE
               WHEN t.state = 'win' THEN -t.amount
               WHEN t.state = 'lost' THEN t.amount
             END) OVER (PARTITION BY t.user_id ORDER BY t.created_at) AS cumulative_change
  FROM odd_transactions_to_cancel t
),
transactions_with_sufficient_balance AS (
  SELECT pt.id, pt.state, pt.user_id, pt.amount
  FROM processable_transactions pt
  JOIN users u ON pt.user_id = u.id
  WHERE u.balance + pt.cumulative_change >= 0
-- Uncomment this line to lock the user row in case of isolation level not serializable
-- to avoid phantom reads
--   FOR UPDATE OF u
),
balance_changes AS (
  SELECT user_id,
         SUM(CASE
               WHEN state = 'win' THEN -amount
               WHEN state = 'lost' THEN amount
             END) AS change
  FROM transactions_with_sufficient_balance
  GROUP BY user_id
),
updated_transactions AS (
  UPDATE transactions
  SET processed = FALSE
  WHERE id IN (SELECT id FROM transactions_with_sufficient_balance)
  RETURNING id, state, user_id, amount
),
updated_users AS (
    UPDATE users
    SET balance = balance + COALESCE (bc.change, 0)
    FROM balance_changes bc
    WHERE users.id = bc.user_id
           RETURNING users.id, users.balance
    ),
final_result AS (
  SELECT uu.id AS user_id, uu.balance AS user_balance, ut.id AS transaction_id, ut.state AS state, ut.amount AS amount
  FROM updated_users uu
  JOIN updated_transactions ut ON uu.id = ut.user_id
)
SELECT user_id, user_balance, transaction_id, state, amount FROM final_result;
`

// CancelOddTransactionsAndUpdateBalance cancels odd transactions and updates user balance.
func (r *TransactionRepositoryImpl) CancelOddTransactionsAndUpdateBalance(ctx context.Context) ([]repositories.CancelOddTransactionsAndUpdateBalanceRow, error) {
	for {
		ids, err := r.processCancelTransaction(ctx, cancelOddTransactions)

		if err == nil {
			return ids, nil
		}

		if isSerializationError(err) {
			// retry transaction if serialization error occurs (SQLSTATE 40001)
			continue
		} else {
			if errors.Is(err, pgx.ErrNoRows) {
				return ids, apperrors.NewInsufficientFundsError()
			}
			return ids, fmt.Errorf("transaction error: %w", err)
		}
	}
}

// processCancelTransaction processes cancel odd transactions and updates user balance.
func (r *TransactionRepositoryImpl) processCancelTransaction(ctx context.Context, query string) ([]repositories.CancelOddTransactionsAndUpdateBalanceRow, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, query)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	defer rows.Close()

	ids := make([]repositories.CancelOddTransactionsAndUpdateBalanceRow, 0)
	for rows.Next() {
		var row repositories.CancelOddTransactionsAndUpdateBalanceRow
		err = rows.Scan(&row.UserId, &row.UserBalance, &row.TransactionId, &row.State, &row.Amount)
		if err != nil {
			tx.Rollback(ctx)
			return nil, err
		}
		ids = append(ids, row)
	}

	err = rows.Err()
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	return ids, nil
}

// GetUserBalance returns users balance.
func (r *TransactionRepositoryImpl) GetUserBalance(ctx context.Context, userId string) (*decimal.Decimal, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	var balance decimal.Decimal
	err = tx.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", userId).Scan(&balance)
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("get user balance: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &balance, nil
}

// GetByTransactionID returns transaction by transaction id.
func (r *TransactionRepositoryImpl) GetByTransactionID(ctx context.Context, transactionID string) (*models.Transaction, error) {
	tx := &models.Transaction{}
	err := r.db.QueryRow(
		ctx,
		"SELECT id, state, amount, source_id, user_id, processed FROM transactions WHERE transaction_id = $1",
		transactionID,
	).Scan(&tx.ID, &tx.State, &tx.Amount, &tx.SourceType.ID, &tx.User.ID, &tx.Processed)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return tx, nil
}

func isSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.SQLState() == repositories.SerializationError
}
