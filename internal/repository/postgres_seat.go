package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type PostgresSeatRepository struct {
	db *pgxpool.Pool
}

func NewPostgresSeatRepository(db *pgxpool.Pool) *PostgresSeatRepository {
	return &PostgresSeatRepository{
		db: db,
	}
}

func (p *PostgresSeatRepository) GetSeatsByShowtime(ctx context.Context, showtimeID int) (*domain.ShowtimeSeats, error) {
	query := `
		SELECT 
			t.id AS theater_id, 
			t.name AS theater_name, 
			h.id AS hall_id, 
			se.id AS seat_id, 
			se.seat_row, 
			se.seat_col, 
			se.seat_type, 
			se.extra_price
		FROM showtimes sh
		JOIN seats se
			ON sh.hall_id = se.hall_id
		JOIN halls h
			ON sh.hall_id = h.id
		JOIN theaters t
			ON h.theater_id = t.id
		WHERE sh.id = $1 AND sh.start_time > NOW()
		ORDER BY se.seat_row, se.seat_col
	`

	rows, err := p.db.Query(ctx, query, showtimeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var showtimeSeats domain.ShowtimeSeats

	for rows.Next() {
		var seat domain.Seat

		seat.Available = true

		err = rows.Scan(
			&showtimeSeats.TheaterID,
			&showtimeSeats.TheaterName,
			&showtimeSeats.HallID,
			&seat.ID,
			&seat.Row,
			&seat.Col,
			&seat.Type,
			&seat.ExtraPrice,
		)
		if err != nil {
			return nil, err
		}

		showtimeSeats.Seats = append(showtimeSeats.Seats, seat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &showtimeSeats, nil
}

func (p *PostgresSeatRepository) GetSeatsByShowtimeAndSeatIds(
	ctx context.Context,
	showtimeID int,
	seatIDs []int) (*domain.ShowtimeSeats, error) {

	query := `
		SELECT 
			t.name,
			m.title,
			h.name,
			sh.base_price,
			sh.start_time,
			se.id, 
			se.seat_row, 
			se.seat_col, 
			se.seat_type, 
			se.extra_price
		FROM showtimes sh
		JOIN seats se
			ON se.hall_id = sh.hall_id
		JOIN halls h
			ON h.id = se.hall_id
		JOIN theaters t
			ON t.id = h.theater_id
		JOIN movies m
			ON m.id = sh.movie_id
		WHERE sh.id = $1 AND se.id = ANY($2::int[]) AND sh.start_time > NOW();
	`

	rows, err := p.db.Query(ctx, query, showtimeID, seatIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var showtimeSeats domain.ShowtimeSeats

	for rows.Next() {
		var seat domain.Seat

		err = rows.Scan(
			&showtimeSeats.TheaterName,
			&showtimeSeats.MovieName,
			&showtimeSeats.HallName,
			&showtimeSeats.Price,
			&showtimeSeats.Date,
			&seat.ID,
			&seat.Row,
			&seat.Col,
			&seat.Type,
			&seat.ExtraPrice,
		)
		if err != nil {
			return nil, err
		}

		showtimeSeats.Seats = append(showtimeSeats.Seats, seat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &showtimeSeats, nil
}
