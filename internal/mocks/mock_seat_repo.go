package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type MockSeatRepo struct {
	GetSeatsByShowtimeFunc func(ctx context.Context, showtimeID int) (*domain.ShowtimeSeats, error)
}

func (m *MockSeatRepo) GetSeatsByShowtime(ctx context.Context, showtimeID int) (*domain.ShowtimeSeats, error) {
	return m.GetSeatsByShowtimeFunc(ctx, showtimeID)
}
