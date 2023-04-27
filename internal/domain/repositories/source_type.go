package repositories

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
)

type SourceTypeRepository interface {
	GetByName(ctx context.Context, name string) (*models.SourceType, error)
}
