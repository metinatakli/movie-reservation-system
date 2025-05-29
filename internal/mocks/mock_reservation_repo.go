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

func (m *MockReservationRepo) GetReservationsSummariesByUserId(
	ctx context.Context,
	userId int,
	pagination domain.Pagination) ([]domain.ReservationSummary, *domain.Metadata, error) {

	args := m.Called(ctx, userId, pagination)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]domain.ReservationSummary), args.Get(1).(*domain.Metadata), args.Error(2)
}

func (m *MockReservationRepo) GetByReservationIdAndUserId(
	ctx context.Context,
	reservationId,
	userId int) (*domain.ReservationDetail, error) {

	args := m.Called(ctx, reservationId, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ReservationDetail), args.Error(1)
}
