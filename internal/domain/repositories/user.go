package repositories

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}
