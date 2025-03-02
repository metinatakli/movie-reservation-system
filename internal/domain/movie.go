package domain

import (
	"context"
	"strings"
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

type Metadata struct {
	CurrentPage  int
	FirstPage    int
	LastPage     int
	PageSize     int
	TotalRecords int
}

func NewMetadata(totalRecords, page, pageSize int) *Metadata {
	return &Metadata{
		CurrentPage:  page,
		FirstPage:    1,
		LastPage:     (totalRecords + pageSize - 1) / pageSize,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
	}
}

type MovieFilters struct {
	Page     int
	PageSize int
	Term     string
	Sort     string
}

func (f MovieFilters) SortColumn() string {
	return strings.TrimPrefix(f.Sort, "-")
}

func (f MovieFilters) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f MovieFilters) Limit() int {
	return f.PageSize
}

func (f MovieFilters) Offset() int {
	return (f.Page - 1) * f.PageSize
}

type MovieRepository interface {
	GetAll(ctx context.Context, filters MovieFilters) ([]*Movie, *Metadata, error)
	GetById(ctx context.Context, id int) (*Movie, error)
}
