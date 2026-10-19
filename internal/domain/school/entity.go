package school

import "context"

type School struct {
	ID   string
	Name string
}

type Repository interface {
	Create(ctx context.Context, school School) error
	FindByID(ctx context.Context, id string) (School, error)
}
