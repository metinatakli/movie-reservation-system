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

type ReservationRepository interface {
	Create(ctx context.Context, reservation Reservation) error
}
