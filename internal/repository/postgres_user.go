package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type PostgesUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) *PostgesUserRepository {
	return &PostgesUserRepository{
		db: db,
	}
}

func (p *PostgesUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (first_name, last_name, email, password_hash, birth_date, gender)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, activated, version`

	err := p.db.QueryRow(ctx,
		query,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password.Hash,
		&user.BirthDate,
		&user.Gender).Scan(&user.ID, &user.CreatedAt, &user.Activated, &user.Version)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return domain.ErrUserAlreadyExists
		}

		return err
	}

	return nil
}
