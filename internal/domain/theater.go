package domain

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Theater struct {
	ID        int
	Name      string
	Address   string
	City      string
	District  string
	Distance  float64
	Amenities []Amenity
	Halls     []Hall
}

type Amenity struct {
	ID          int
	Name        string
	Description string
}

type Hall struct {
	ID        int
	TheaterID int
	Name      string
	Amenities []Amenity
	Showtimes []Showtime
}

type Showtime struct {
	ID        int
	StartTime time.Time
	BasePrice pgtype.Numeric
}

type TheaterRepository interface {
	GetTheatersByMovieAndLocationAndDate(
		ctx context.Context,
		movieID int,
		date time.Time,
		lat, long float64,
		pagination Pagination,
	) ([]Theater, *Metadata, error)
}
