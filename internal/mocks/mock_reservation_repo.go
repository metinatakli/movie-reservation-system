package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockReservationRepo struct {
	mock.Mock
	domain.ReservationRepository
}

func (m *MockReservationRepo) Create(ctx context.Context, reservation domain.Reservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *MockReservationRepo) GetSeatsByShowtimeId(ctx context.Context, showtimeId int) ([]domain.ReservationSeat, error) {
	args := m.Called(ctx, showtimeId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.ReservationSeat), args.Error(1)
}
