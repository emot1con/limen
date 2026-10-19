package school

import (
	"context"
	"fmt"

	schooldomain "limen/internal/domain/school"
)

type CreateInput struct {
	ID   string
	Name string
}

type CreateUseCase struct {
	repository schooldomain.Repository
}

func NewCreateUseCase(repository schooldomain.Repository) CreateUseCase {
	return CreateUseCase{repository: repository}
}

func (uc CreateUseCase) Execute(ctx context.Context, input CreateInput) (schooldomain.School, error) {
	if input.Name == "" {
		return schooldomain.School{}, fmt.Errorf("school name is required")
	}

	school := schooldomain.School{
		ID:   input.ID,
		Name: input.Name,
	}

	if err := uc.repository.Create(ctx, school); err != nil {
		return schooldomain.School{}, err
	}

	return school, nil
}
