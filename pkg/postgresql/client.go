package postgresql

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mufasadev/enlabs-test/pkg/util/repeat"
	"time"
)

const ClientTimeout = 5 * time.Second

type Client interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewClient(cfg *pgxpool.Config, MaxConnAttempts int) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	err = repeat.Repeat(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), ClientTimeout)
		defer cancel()

		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return err
		}

		err = pool.Ping(ctx)
		if err != nil {
			return err
		}

		return nil
	}, MaxConnAttempts, ClientTimeout)

	if err != nil {
		return nil, err
	}

	return pool, err
}
