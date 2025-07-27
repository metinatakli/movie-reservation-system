package mocks

import (
	"context"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type MockMovieRepo struct {
	domain.MovieRepository
	GetAllFunc     func(ctx context.Context, filters domain.Pagination) ([]*domain.Movie, *domain.Metadata, error)
	GetByIdFunc    func(ctx context.Context, id int) (*domain.Movie, error)
	ExistsByIdFunc func(ctx context.Context, id int) (bool, error)
}

func (m *MockMovieRepo) GetAll(ctx context.Context, filters domain.Pagination) ([]*domain.Movie, *domain.Metadata, error) {
	return m.GetAllFunc(ctx, filters)
}

func (m *MockMovieRepo) GetById(ctx context.Context, id int) (*domain.Movie, error) {
	return m.GetByIdFunc(ctx, id)
}

func (m *MockMovieRepo) ExistsById(ctx context.Context, id int) (bool, error) {
	return m.ExistsByIdFunc(ctx, id)
}
