package repositories

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
	"github.com/mufasadev/enlabs-test/internal/domain/repositories"
	"strings"
)

type SourceTypeRepositoryImpl struct {
	db *pgxpool.Pool
}

func NewSourceTypeRepositoryImpl(db *pgxpool.Pool) repositories.SourceTypeRepository {
	return &SourceTypeRepositoryImpl{
		db: db,
	}
}

func (r *SourceTypeRepositoryImpl) GetByName(ctx context.Context, name string) (*models.SourceType, error) {
	sourceType := &models.SourceType{}
	err := r.db.QueryRow(
		ctx,
		"SELECT id, name FROM sources WHERE name = $1",
		strings.ToLower(name),
	).Scan(&sourceType.ID, &sourceType.Name)

	if err != nil {
		return nil, err
	}

	return sourceType, nil
}
