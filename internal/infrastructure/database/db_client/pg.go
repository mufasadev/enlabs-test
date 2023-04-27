package db_client

import (
	"context"
	"fmt"
	decimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mufasadev/enlabs-test/internal/config"
	"github.com/mufasadev/enlabs-test/pkg/postgresql"
	"strconv"
)

type PGClient struct {
	cfg config.PostgreSQL
}

func NewPGClient(cfg config.PostgreSQL) *PGClient {
	return &PGClient{cfg: cfg}
}

// Connect connects to the database and returns a pgxpool.Pool.
func (c *PGClient) Connect() (*pgxpool.Pool, error) {
	pgxConfig, err := pgxpool.ParseConfig(c.cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("pgxpool.ParseConfig: %w", err)
	}

	// Register decimal type
	pgxConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		decimal.Register(conn.TypeMap())
		return nil
	}

	maxAttempts, err := strconv.Atoi(c.cfg.MaxConnAttempts)
	if err != nil {
		return nil, fmt.Errorf("strconv.Atoi: %w", err)
	}

	db, err := postgresql.NewClient(pgxConfig, maxAttempts)
	if err != nil {
		return nil, fmt.Errorf("postgresql.NewClient: %w", err)
	}

	return db, nil
}
