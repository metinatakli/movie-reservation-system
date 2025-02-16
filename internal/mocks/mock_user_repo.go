package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type MockUserRepo struct {
	domain.UserRepository
	CreateFunc func(ctx context.Context, user *domain.User) error
}

func (m *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.CreateFunc(ctx, user)
}
