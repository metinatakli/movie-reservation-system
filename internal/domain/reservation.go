package domain

import (
	"context"
	"time"
)

type Reservation struct {
	ID                int
	UserID            int
	ShowtimeID        int
	CheckoutSessionID string
	PaymentID         int
	ReservationSeats  []ReservationSeat
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ReservationSeat struct {
	ReservationID int
	ShowtimeID    int
	SeatID        int
}

type ReservationSummary struct {
	ReservationID  int
	MovieTitle     string
	MoviePosterUrl string
	ShowtimeDate   time.Time
	TheaterName    string
	HallName       string
	CreatedAt      time.Time
}

type ReservationRepository interface {
	Create(ctx context.Context, reservation Reservation) error
	GetSeatsByShowtimeId(ctx context.Context, showtimeId int) ([]ReservationSeat, error)
	GetReservationsSummariesByUserId(ctx context.Context, userId int, pagination Pagination) ([]ReservationSummary, *Metadata, error)
}
