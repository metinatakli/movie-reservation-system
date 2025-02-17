package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

// MockTokenRepo is a mock implementation of TokenRepository
type MockTokenRepo struct {
	domain.TokenRepository
	CreateFunc           func(ctx context.Context, token *domain.Token) error
	DeleteAllForUserFunc func(ctx context.Context, tokenScope string, userID int) error
}

func (m *MockTokenRepo) Create(ctx context.Context, token *domain.Token) error {
	return m.CreateFunc(ctx, token)
}

func (m *MockTokenRepo) DeleteAllForUser(ctx context.Context, tokenScope string, userID int) error {
	return m.DeleteAllForUserFunc(ctx, tokenScope, userID)
}
