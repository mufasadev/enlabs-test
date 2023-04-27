package interactor

import (
	"context"
	"github.com/mufasadev/enlabs-test/internal/domain/repositories"
)

type SourceTypeInteractor struct {
	sourceTypeRepository repositories.SourceTypeRepository
}

func NewSourceTypeInteractor(Repository repositories.SourceTypeRepository) *SourceTypeInteractor {
	return &SourceTypeInteractor{sourceTypeRepository: Repository}
}

func (s *SourceTypeInteractor) ExistsByName(ctx context.Context, name string) (bool, error) {
	_, err := s.sourceTypeRepository.GetByName(ctx, name)
	if err != nil {
		return false, err
	}
	return true, nil
}
