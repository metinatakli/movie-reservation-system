package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type PostgresTokenRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTokenRepository(db *pgxpool.Pool) *PostgresTokenRepository {
	return &PostgresTokenRepository{
		db: db,
	}
}

func (p *PostgresTokenRepository) Create(ctx context.Context, token *domain.Token) error {
	query := `INSERT INTO tokens (hash, user_id, expiry, scope)
			VALUES($1, $2, $3, $4)
			ON CONFLICT ON CONSTRAINT unique_user_scope DO 
			UPDATE SET
				hash = EXCLUDED.hash,  
				expiry = EXCLUDED.expiry`

	_, err := p.db.Exec(ctx, query, token.Hash, token.UserId, token.Expiry, token.Scope)

	return err
}

func (p *PostgresTokenRepository) DeleteAllForUser(ctx context.Context, tokenScope string, userID int) error {
	query := `DELETE FROM tokens
			WHERE scope = $1 AND user_id = $2`

	_, err := p.db.Exec(ctx, query, tokenScope, userID)

	return err
}
