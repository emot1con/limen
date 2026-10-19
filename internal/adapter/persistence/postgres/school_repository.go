package postgres

import (
	"context"
	"sync"

	"limen/internal/domain/school"
)

type SchoolRepository struct {
	mu      sync.RWMutex
	schools map[string]school.School
}

func NewSchoolRepository() *SchoolRepository {
	return &SchoolRepository{schools: make(map[string]school.School)}
}

func (r *SchoolRepository) Create(_ context.Context, school school.School) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.schools[school.ID] = school
	return nil
}

func (r *SchoolRepository) FindByID(_ context.Context, id string) (school.School, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.schools[id], nil
}
