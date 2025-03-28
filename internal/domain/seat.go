package domain

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type ShowtimeSeats struct {
	TheaterID   int
	TheaterName string
	HallID      int
	Seats       []Seat
	Price       pgtype.Numeric
}

type Seat struct {
	ID         int
	Row        int
	Col        int
	Type       string
	ExtraPrice pgtype.Numeric
}

type SeatRepository interface {
	GetSeatsByShowtime(ctx context.Context, showtimeID int) (*ShowtimeSeats, error)
	GetSeatsByShowtimeAndSeatIds(ctx context.Context, showtimeID int, seatIDs []int) (*ShowtimeSeats, error)
}
