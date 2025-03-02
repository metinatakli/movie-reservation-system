package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type PostgresMovieRepository struct {
	db *pgxpool.Pool
}

func NewPostgresMovieRepository(db *pgxpool.Pool) *PostgresMovieRepository {
	return &PostgresMovieRepository{
		db: db,
	}
}

func (p *PostgresMovieRepository) GetAll(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error) {
	query := fmt.Sprintf(`SELECT count(*) OVER(), id, title, description, release_date, poster_url
		FROM movies
		WHERE ((to_tsvector('english', title) @@ plainto_tsquery('english', $1) 
			OR to_tsvector('english', description) @@ plainto_tsquery('english', $1))
			OR $1 = '') 
		ORDER BY %s %s
		LIMIT $2 OFFSET $3`, filters.SortColumn(), filters.SortDirection())

	rows, err := p.db.Query(ctx, query, filters.Term, filters.Limit(), filters.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	totalRecords := 0
	movies := []*domain.Movie{}

	for rows.Next() {
		var movie domain.Movie

		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.Title,
			&movie.Description,
			&movie.ReleaseDate,
			&movie.PosterUrl,
		)

		if err != nil {
			return nil, nil, err
		}

		movies = append(movies, &movie)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	metadata := domain.NewMetadata(totalRecords, filters.Page, filters.PageSize)

	return movies, metadata, nil
}
