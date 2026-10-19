package user

import "context"

type User struct {
	ID    string
	Name  string
	Email string
}

type Repository interface {
	Create(ctx context.Context, user User) error
	FindByID(ctx context.Context, id string) (User, error)
}
