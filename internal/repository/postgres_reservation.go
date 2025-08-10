package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
			SET status = 'completed', stripe_checkout_session_id = $1, payment_date = NOW(), updated_at = NOW()
			WHERE id = $2 AND status = 'pending'
		`

		cmdTag, err := tx.Exec(ctx, query, reservation.CheckoutSessionID, reservation.PaymentID)
		if err != nil {
			return err
		}

		if cmdTag.RowsAffected() != 1 {
			return fmt.Errorf(
				"failed to update payment: record not found or status was not pending (payment_id: %d)",
				reservation.PaymentID,
			)
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

func (p *PostgresReservationRepository) GetByReservationIdAndUserId(
	ctx context.Context,
	reservationId,
	userId int) (*domain.ReservationDetail, error) {

	query := `
		SELECT
			r.id,
			m.title,
			m.poster_url,
			s.start_time,
			t.name,
			h.name,
			r.created_at,
			p.amount,
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object(
					'row', s.seat_row, 
					'col', s.seat_col, 
					'type', s.seat_type)), '[]')
				FROM reservation_seats rs
				JOIN seats s ON rs.seat_id = s.id
				WHERE rs.reservation_id = r.id
			) AS seats,
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object(
					'id', a.id, 
					'name', a.name, 
					'description', a.description)), '[]')
				FROM hall_amenities ha
				JOIN amenities a ON ha.amenity_id = a.id
				WHERE ha.hall_id = h.id
			) AS hall_amenities,
			(
				SELECT COALESCE(jsonb_agg(jsonb_build_object(
					'id', a.id, 
					'name', a.name, 
					'description', a.description)), '[]')
				FROM theater_amenities ta
				JOIN amenities a ON ta.amenity_id = a.id
				WHERE ta.theater_id = t.id
			) AS theater_amenities
		FROM reservations r
		JOIN payments p ON r.payment_id = p.id
		JOIN showtimes s ON r.showtime_id = s.id
		JOIN movies m ON s.movie_id = m.id
		JOIN halls h ON s.hall_id = h.id
		JOIN theaters t ON h.theater_id = t.id
		WHERE r.id = $1 AND r.user_id = $2
		GROUP BY r.id, p.id, s.id, m.id, h.id, t.id
	`

	var reservationDetail domain.ReservationDetail
	var seatsJson, hallAmenitiesJson, theaterAmenitiesJson json.RawMessage

	err := p.db.QueryRow(ctx, query, reservationId, userId).Scan(
		&reservationDetail.ReservationID,
		&reservationDetail.MovieTitle,
		&reservationDetail.MoviePosterUrl,
		&reservationDetail.ShowtimeDate,
		&reservationDetail.TheaterName,
		&reservationDetail.HallName,
		&reservationDetail.CreatedAt,
		&reservationDetail.TotalPrice,
		&seatsJson,
		&hallAmenitiesJson,
		&theaterAmenitiesJson,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	if err := json.Unmarshal(seatsJson, &reservationDetail.Seats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reservation seats: %w", err)
	}

	if err := json.Unmarshal(hallAmenitiesJson, &reservationDetail.HallAmenities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hall amenities: %w", err)
	}

	if err := json.Unmarshal(theaterAmenitiesJson, &reservationDetail.TheaterAmenities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal theater amenities: %w", err)
	}

	return &reservationDetail, nil
}
