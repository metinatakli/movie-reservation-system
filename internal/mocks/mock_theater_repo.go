package mocks

import (
	"context"
	"time"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type MockTheaterRepo struct {
	GetTheatersByMovieAndLocationAndDateFunc func(
		context.Context,
		int,
		time.Time,
		float64,
		float64,
		domain.Pagination) ([]domain.Theater, *domain.Metadata, error)
}

func (m *MockTheaterRepo) GetTheatersByMovieAndLocationAndDate(
	ctx context.Context,
	movieID int,
	date time.Time,
	longitude, latitude float64,
	pagination domain.Pagination) ([]domain.Theater, *domain.Metadata, error) {

	return m.GetTheatersByMovieAndLocationAndDateFunc(ctx, movieID, date, longitude, latitude, pagination)
}
