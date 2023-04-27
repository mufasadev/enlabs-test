package repositories

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mufasadev/enlabs-test/internal/domain/models"
	"github.com/mufasadev/enlabs-test/internal/domain/repositories"
	"github.com/mufasadev/enlabs-test/internal/errors"
)

type UserRepositoryImpl struct {
	db *pgxpool.Pool
}

func NewUserRepositoryImpl(db *pgxpool.Pool) repositories.UserRepository {
	return &UserRepositoryImpl{
		db: db,
	}
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		ctx,
		"SELECT id, balance FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Balance)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.NewBadRequestError("User not found")
		}
		return nil, err
	}

	return user, nil
}

func (r *UserRepositoryImpl) Update(ctx context.Context, user *models.User) error {
	_, err := r.db.Exec(
		ctx,
		"UPDATE users SET balance = $1 WHERE id = $2",
		user.Balance,
		user.ID,
	)
	return err
}
