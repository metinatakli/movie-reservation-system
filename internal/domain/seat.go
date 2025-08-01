package domain

import (
	"context"
	"time"
)

type ShowtimeSeats struct {
	TheaterID   int
	TheaterName string
	MovieName   string
	HallName    string
	Date        time.Time
	HallID      int
	Seats       []Seat
	Price       float64
}

type Seat struct {
	ID         int
	Row        int
	Col        int
	Type       string
	ExtraPrice float64
	Available  bool
}

type SeatRepository interface {
	GetSeatsByShowtime(ctx context.Context, showtimeID int) (*ShowtimeSeats, error)
	GetSeatsByShowtimeAndSeatIds(ctx context.Context, showtimeID int, seatIDs []int) (*ShowtimeSeats, error)
}
