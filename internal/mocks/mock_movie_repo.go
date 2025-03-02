package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type MockMovieRepo struct {
	domain.MovieRepository
	GetAllFunc  func(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error)
	GetByIdFunc func(ctx context.Context, id int) (*domain.Movie, error)
}

func (m *MockMovieRepo) GetAll(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error) {
	return m.GetAllFunc(ctx, filters)
}

func (m *MockMovieRepo) GetById(ctx context.Context, id int) (*domain.Movie, error) {
	return m.GetByIdFunc(ctx, id)
}
