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

func (p *PostgresReservationRepository) GetSeatsByShowtimeId(
	ctx context.Context,
	showtimeId int) ([]domain.ReservationSeat, error) {

	query := `
		SELECT reservation_id, showtime_id, seat_id
		FROM reservation_seats
		WHERE showtime_id = $1
	`

	rows, err := p.db.Query(ctx, query, showtimeId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reservationSeats := make([]domain.ReservationSeat, 0)

	for rows.Next() {
		var reservationSeat domain.ReservationSeat

		err = rows.Scan(
			&reservationSeat.ReservationID,
			&reservationSeat.ShowtimeID,
			&reservationSeat.SeatID,
		)

		if err != nil {
			return nil, err
		}

		reservationSeats = append(reservationSeats, reservationSeat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return reservationSeats, nil
}

func (p *PostgresReservationRepository) GetReservationsSummariesByUserId(
	ctx context.Context,
	userId int,
	pagination domain.Pagination) ([]domain.ReservationSummary, *domain.Metadata, error) {

	query := `
		SELECT
			COUNT(*) OVER(),
			r.id,
			m.title,
			m.poster_url,
			s.start_time,
			t.name,
			h.name,
			r.created_at
		FROM reservations r
		JOIN showtimes s ON r.showtime_id = s.id
		JOIN movies m ON s.movie_id = m.id
		JOIN halls h ON s.hall_id = h.id
		JOIN theaters t ON h.theater_id = t.id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := p.db.Query(ctx, query, userId, pagination.Limit(), pagination.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	reservations := make([]domain.ReservationSummary, 0)
	totalRecords := 0

	for rows.Next() {
		var reservation domain.ReservationSummary

		err := rows.Scan(
			&totalRecords,
			&reservation.ReservationID,
			&reservation.MovieTitle,
			&reservation.MoviePosterUrl,
			&reservation.ShowtimeDate,
			&reservation.TheaterName,
			&reservation.HallName,
			&reservation.CreatedAt,
		)
		if err != nil {
			return nil, nil, err
		}

		reservations = append(reservations, reservation)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	metadata := domain.NewMetadata(totalRecords, pagination.Page, pagination.PageSize)

	return reservations, metadata, nil
}
