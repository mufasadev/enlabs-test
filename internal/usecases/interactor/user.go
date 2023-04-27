package interactor

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/domain/repositories"
)

type UserInteractor struct {
	userRepository repositories.UserRepository
}

func NewUserInteractor(Repository repositories.UserRepository) *UserInteractor {
	return &UserInteractor{userRepository: Repository}
}

func (u *UserInteractor) ExistsByID(ctx context.Context, id string) (bool, error) {
	_, err := u.userRepository.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (u *UserInteractor) GetBalance(ctx context.Context, id string) (float64, error) {
	user, err := u.userRepository.GetByID(ctx, id)
	if err != nil {
		return 0.0, err
	}
	b, _ := user.Balance.Float64()
	return b, nil
}
