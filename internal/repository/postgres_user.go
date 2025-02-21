package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
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

func (p *PostgesUserRepository) GetByToken(ctx context.Context, tokenHash []byte, tokenScope string) (*domain.User, error) {
	query := `SELECT u.id, u.first_name, u.last_name, u.birth_date, u.gender, u.email, u.password_hash, u.activated, u.version
		FROM users u
		INNER JOIN tokens t ON u.id = t.user_id
		WHERE t.hash = $1 AND t.scope = $2 AND t.expiry > $3`

	user := &domain.User{}

	err := p.db.QueryRow(ctx, query, tokenHash, tokenScope, time.Now()).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.BirthDate,
		&user.Gender,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	return user, nil
}

func (p *PostgesUserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET first_name    = COALESCE($3, first_name),
			last_name     = COALESCE($4, last_name),
			password_hash = COALESCE($5, password_hash),
			birth_date    = COALESCE($6, birth_date),
			gender        = COALESCE($7, gender),
			activated     = COALESCE($8, activated),
			updated_at    = NOW(),
			version       = version + 1
		WHERE id = $1 AND version = $2
		RETURNING version`

	args := []any{user.ID,
		user.Version,
		user.FirstName,
		user.LastName,
		user.Password.Hash,
		user.BirthDate,
		user.Gender,
		user.Activated}

	err := p.db.QueryRow(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return domain.ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (p *PostgesUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, password_hash 
		FROM users
		WHERE email = $1 AND activated = true AND is_active = true`

	user := &domain.User{}

	err := p.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Password.Hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	return user, nil
}

func (p *PostgesUserRepository) GetById(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT id, first_name, last_name, birth_date, gender, email, activated, version, created_at
		FROM users
		WHERE id = $1 AND activated = true AND is_active = true`

	user := &domain.User{}

	err := p.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.BirthDate,
		&user.Gender,
		&user.Email,
		&user.Activated,
		&user.Version,
		&user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	return user, nil
}
