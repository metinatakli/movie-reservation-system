package domain

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Movie struct {
	ID          int
	Title       string
	Description string
	Genres      []string
	Language    string
	ReleaseDate time.Time
	Duration    int
	PosterUrl   string
	Director    string
	CastMembers []string
	Rating      pgtype.Numeric
}

type MovieRepository interface {
	GetAll(ctx context.Context, pagination Pagination) ([]*Movie, *Metadata, error)
	GetById(ctx context.Context, id int) (*Movie, error)
}
