package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type MockUserRepo struct {
	domain.UserRepository
	CreateFunc     func(ctx context.Context, user *domain.User) error
	GetByTokenFunc func(ctx context.Context, hash []byte, scope string) (*domain.User, error)
	UpdateFunc     func(ctx context.Context, user *domain.User) error
	GetByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	GetByIdFunc    func(ctx context.Context, id int) (*domain.User, error)
}

func (m *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.CreateFunc(ctx, user)
}

func (m *MockUserRepo) GetByToken(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
	return m.GetByTokenFunc(ctx, hash, scope)
}

func (m *MockUserRepo) Update(ctx context.Context, user *domain.User) error {
	return m.UpdateFunc(ctx, user)
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.GetByEmailFunc(ctx, email)
}

func (m *MockUserRepo) GetById(ctx context.Context, id int) (*domain.User, error) {
	return m.GetByIdFunc(ctx, id)
}
