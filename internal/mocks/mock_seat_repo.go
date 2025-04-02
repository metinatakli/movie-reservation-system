package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockSeatRepo struct {
	mock.Mock
	domain.SeatRepository
}

func (m *MockSeatRepo) GetSeatsByShowtimeAndSeatIds(ctx context.Context, showtimeID int, seatIDs []int) (*domain.ShowtimeSeats, error) {
	args := m.Called(ctx, showtimeID, seatIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ShowtimeSeats), args.Error(1)
}

func (m *MockSeatRepo) GetSeatsByShowtime(ctx context.Context, showtimeID int) (*domain.ShowtimeSeats, error) {
	args := m.Called(ctx, showtimeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ShowtimeSeats), args.Error(1)
}
