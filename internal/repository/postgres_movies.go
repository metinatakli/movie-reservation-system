package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (p *PostgresMovieRepository) GetAll(
	ctx context.Context,
	pagination domain.Pagination) ([]*domain.Movie, *domain.Metadata, error) {

	query := fmt.Sprintf(`SELECT count(*) OVER(), id, title, description, release_date, poster_url
		FROM movies
		WHERE ((to_tsvector('english', title) @@ plainto_tsquery('english', $1) 
			OR to_tsvector('english', description) @@ plainto_tsquery('english', $1))
			OR $1 = '') 
		ORDER BY %s %s
		LIMIT $2 OFFSET $3`, pagination.SortColumn(), pagination.SortDirection())

	rows, err := p.db.Query(ctx, query, pagination.Term, pagination.Limit(), pagination.Offset())
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

	metadata := domain.NewMetadata(totalRecords, pagination.Page, pagination.PageSize)

	return movies, metadata, nil
}

func (p *PostgresMovieRepository) GetById(ctx context.Context, id int) (*domain.Movie, error) {
	query := `SELECT id, title, description, genres, language, release_date, duration, poster_url, director,
	 cast_members, rating
		FROM movies
		WHERE id = $1`

	movie := &domain.Movie{}

	err := p.db.QueryRow(ctx, query, id).Scan(
		&movie.ID,
		&movie.Title,
		&movie.Description,
		&movie.Genres,
		&movie.Language,
		&movie.ReleaseDate,
		&movie.Duration,
		&movie.PosterUrl,
		&movie.Director,
		&movie.CastMembers,
		&movie.Rating)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	return movie, nil
}

func (p *PostgresMovieRepository) ExistsById(ctx context.Context, id int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM movies WHERE id = $1)`

	var exists bool
	err := p.db.QueryRow(ctx, query, id).Scan(&exists)

	return exists, err
}
