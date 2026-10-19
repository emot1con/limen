package user

import (
	"context"
	"fmt"

	userdomain "limen/internal/domain/user"
)

type CreateInput struct {
	ID    string
	Name  string
	Email string
}

type CreateUseCase struct {
	repository userdomain.Repository
}

func NewCreateUseCase(repository userdomain.Repository) CreateUseCase {
	return CreateUseCase{repository: repository}
}

func (uc CreateUseCase) Execute(ctx context.Context, input CreateInput) (userdomain.User, error) {
	if input.Name == "" || input.Email == "" {
		return userdomain.User{}, fmt.Errorf("name and email are required")
	}

	user := userdomain.User{
		ID:    input.ID,
		Name:  input.Name,
		Email: input.Email,
	}

	if err := uc.repository.Create(ctx, user); err != nil {
		return userdomain.User{}, err
	}

	return user, nil
}
