package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type PostgresReservationRepository struct {
	db *pgxpool.Pool
}

func NewPostgresReservationRepository(db *pgxpool.Pool) *PostgresReservationRepository {
	return &PostgresReservationRepository{
		db: db,
	}
}

func (p *PostgresReservationRepository) Create(ctx context.Context, reservation domain.Reservation) error {
	return runInTx(ctx, p.db, func(tx pgx.Tx) error {
		query := `
			UPDATE payments
			SET status = 'completed', payment_date = NOW(), updated_at = NOW()
			WHERE stripe_checkout_session_id = $1
			RETURNING id
		`

		err := tx.QueryRow(ctx, query, reservation.CheckoutSessionID).Scan(&reservation.PaymentID)
		if err != nil {
			return err
		}

		query = `
			INSERT INTO reservations (user_id, showtime_id, payment_id)
			VALUES ($1, $2, $3)
			RETURNING id
		`

		err = tx.QueryRow(
			ctx,
			query,
			reservation.UserID,
			reservation.ShowtimeID,
			reservation.PaymentID).Scan(&reservation.ID)

		if err != nil {
			return err
		}

		rows := make([][]any, 0, len(reservation.ReservationSeats))
		for _, seat := range reservation.ReservationSeats {
			rows = append(rows, []any{
				reservation.ID,
				reservation.ShowtimeID,
				seat.SeatID,
			})
		}

		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"reservation_seats"},
			[]string{"reservation_id", "showtime_id", "seat_id"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return err
		}

		return nil
	})
}

func runInTx(ctx context.Context, db *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	var txOptions pgx.TxOptions

	tx, err := db.BeginTx(ctx, txOptions)
	if err != nil {
		return err
	}

	err = fn(tx)
	if err == nil {
		return tx.Commit(ctx)
	}

	rollbackErr := tx.Rollback(ctx)
	if rollbackErr != nil {
		return errors.Join(err, rollbackErr)
	}

	return err
}
