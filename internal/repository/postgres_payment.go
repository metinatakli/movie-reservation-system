package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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

func (p *PostgresPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (
			user_id, 
			amount, 
			currency,
			status
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := p.db.QueryRow(
		ctx,
		query,
		payment.UserID,
		payment.Amount,
		payment.Currency,
		payment.Status,
	).Scan(&payment.ID)

	return err
}

func (p *PostgresPaymentRepository) GetById(ctx context.Context, id int) (*domain.Payment, error) {
	query := `
		SELECT id, user_id, stripe_checkout_session_id, amount, currency, status, error_message, 
			payment_date, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	payment := &domain.Payment{}
	err := p.db.QueryRow(ctx, query, id).Scan(
		&payment.ID,
		&payment.UserID,
		&payment.CheckoutSessionId,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.ErrorMsg,
		&payment.PaymentDate,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	return payment, nil
}

func (p *PostgresPaymentRepository) UpdateStatus(
	ctx context.Context,
	checkoutSessionID string,
	status domain.PaymentStatus,
	errMsg string) error {

	query := `UPDATE payments
		SET status = $1, error_message = $2
		WHERE stripe_checkout_session_id = $3
	`

	_, err := p.db.Exec(ctx, query, status, errMsg, checkoutSessionID)
	return err
}
