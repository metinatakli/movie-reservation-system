package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockPaymentRepo struct {
	mock.Mock
	domain.PaymentRepository
}

func (m *MockPaymentRepo) Create(ctx context.Context, payment domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}
