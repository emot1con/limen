package postgres

import (
	"context"
	"sync"

	"limen/internal/domain/user"
)

type UserRepository struct {
	mu    sync.RWMutex
	users map[string]user.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{users: make(map[string]user.User)}
}

func (r *UserRepository) Create(_ context.Context, user user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = user
	return nil
}

func (r *UserRepository) FindByID(_ context.Context, id string) (user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.users[id], nil
}
