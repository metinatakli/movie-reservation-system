package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type MockMovieRepo struct {
	domain.MovieRepository
	GetAllFunc func(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error)
}

func (m *MockMovieRepo) GetAll(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error) {
	return m.GetAllFunc(ctx, filters)
}
