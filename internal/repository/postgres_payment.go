package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type PostgresPaymentRepository struct {
	db *pgxpool.Pool
}

func NewPostgresPaymentRepository(db *pgxpool.Pool) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{
		db: db,
	}
}

func (p *PostgresPaymentRepository) Create(ctx context.Context, payment domain.Payment) error {
	query := `
		INSERT INTO payments (
			user_id, 
			stripe_checkout_session_id, 
			amount, 
			currency,
			status, 
			error_message, 
			payment_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := p.db.Exec(
		ctx,
		query,
		payment.UserID,
		payment.CheckoutSessionId,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.ErrorMsg,
		payment.PaymentDate,
	)

	return err
}
