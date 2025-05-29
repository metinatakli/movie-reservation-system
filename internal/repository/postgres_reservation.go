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
			h.id,
			t.id
		FROM reservations r
		JOIN payments p ON r.payment_id = p.id
		JOIN showtimes s ON r.showtime_id = s.id
		JOIN movies m ON s.movie_id = m.id
		JOIN halls h ON s.hall_id = h.id
		JOIN theaters t ON h.theater_id = t.id
		WHERE r.id = $1 AND r.user_id = $2
	`

	var reservationDetail domain.ReservationDetail
	var theaterId int
	var hallId int

	err := p.db.QueryRow(ctx, query, reservationId, userId).Scan(
		&reservationDetail.ReservationID,
		&reservationDetail.MovieTitle,
		&reservationDetail.MoviePosterUrl,
		&reservationDetail.ShowtimeDate,
		&reservationDetail.TheaterName,
		&reservationDetail.HallName,
		&reservationDetail.CreatedAt,
		&reservationDetail.TotalPrice,
		&theaterId,
		&hallId,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}

		return nil, err
	}

	reservationSeats, err := p.retrieveReservationSeats(ctx, reservationId)
	if err != nil {
		return nil, err
	}

	theaterAmenities, err := p.retrieveTheaterAmenities(ctx, theaterId)
	if err != nil {
		return nil, err
	}

	hallAmenities, err := p.retrieveHallAmenities(ctx, hallId)
	if err != nil {
		return nil, err
	}

	reservationDetail.Seats = reservationSeats
	reservationDetail.TheaterAmenities = theaterAmenities
	reservationDetail.HallAmenities = hallAmenities

	return &reservationDetail, nil
}

func (p *PostgresReservationRepository) retrieveTheaterAmenities(
	ctx context.Context, theaterId int) ([]domain.Amenity, error) {

	query := `
		SELECT a.id, a.name, a.description
		FROM amenities a
		JOIN theater_amenities ta 
			ON a.id = ta.amenity_id AND ta.theater_id = $1
	`

	rows, err := p.db.Query(ctx, query, theaterId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	amenities := make([]domain.Amenity, 0)

	for rows.Next() {
		var amenity domain.Amenity

		err := rows.Scan(&amenity.ID, &amenity.Name, &amenity.Description)
		if err != nil {
			return nil, err
		}

		amenities = append(amenities, amenity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return amenities, err
}

func (p *PostgresReservationRepository) retrieveHallAmenities(
	ctx context.Context, hallId int) ([]domain.Amenity, error) {

	query := `
		SELECT a.id, a.name, a.description
		FROM amenities a
		JOIN hall_amenities ha 
			ON a.id = ha.amenity_id AND ha.hall_id = $1
	`

	rows, err := p.db.Query(ctx, query, hallId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	amenities := make([]domain.Amenity, 0)

	for rows.Next() {
		var amenity domain.Amenity

		err := rows.Scan(&amenity.ID, &amenity.Name, &amenity.Description)
		if err != nil {
			return nil, err
		}

		amenities = append(amenities, amenity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return amenities, err
}

func (p *PostgresReservationRepository) retrieveReservationSeats(
	ctx context.Context,
	reservationId int) ([]domain.ReservationDetailSeat, error) {

	query := `
		SELECT s.seat_row, s.seat_col, s.seat_type
		FROM reservation_seats rs
		JOIN seats s ON rs.seat_id = s.id
		WHERE reservation_id = $1
	`

	rows, err := p.db.Query(ctx, query, reservationId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reservationSeats := make([]domain.ReservationDetailSeat, 0)

	for rows.Next() {
		var reservationSeat domain.ReservationDetailSeat

		err := rows.Scan(
			&reservationSeat.Row,
			&reservationSeat.Col,
			&reservationSeat.Type,
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
