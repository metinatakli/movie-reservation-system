package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

type PostgresTheaterRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTheaterRepository(db *pgxpool.Pool) *PostgresTheaterRepository {
	return &PostgresTheaterRepository{
		db: db,
	}
}

func (p *PostgresTheaterRepository) GetTheatersByMovieAndLocationAndDate(
	ctx context.Context,
	movieID int,
	date time.Time,
	long, lat float64,
	pagination domain.Pagination,
) ([]domain.Theater, *domain.Metadata, error) {
	query := `
		WITH 
		movie_halls AS (
			SELECT 
				h.id,
				h.theater_id AS theaterID, 
				h.name, 
				COALESCE(jsonb_agg(
					DISTINCT jsonb_build_object(
						'id', a.id,
						'name', a.name,
						'description', a.description
					)) FILTER (WHERE a.id IS NOT NULL), '[]') AS amenities,
				COALESCE(jsonb_agg(
					DISTINCT jsonb_build_object(
						'id', s.id,
						'startTime', s.start_time,
						'basePrice', s.base_price
					)), '[]') AS showtimes
			FROM halls h
			INNER JOIN showtimes s 
				ON s.hall_id = h.id 
				AND s.movie_id = $1
				AND s.start_time::date = $2
			LEFT JOIN hall_amenities ha ON ha.hall_id = h.id
			LEFT JOIN amenities a ON ha.amenity_id = a.id
			GROUP BY h.id, h.theater_id, h.name
		)
		SELECT 
			t.id, 
			t.name, 
			t.address, 
			t.city,
			t.district,
			ST_Distance(t.location, ST_SetSRID(ST_MakePoint($3, $4), 4326)) / 1000 AS distance,
			COALESCE(ta.amenities, '[]') AS amenities,
			mh.halls,
			COUNT(*) OVER() AS totalCount
		FROM theaters t
		INNER JOIN (
			SELECT mh.theaterID, jsonb_agg(mh) AS halls
			FROM movie_halls mh
			GROUP BY mh.theaterID
		) mh ON mh.theaterID = t.id
		LEFT JOIN LATERAL (
			SELECT jsonb_agg(
				json_build_object(
					'id', a.id,
					'name', a.name,
					'description', a.description
				)
			) AS amenities
			FROM theater_amenities ta
			LEFT JOIN amenities a ON ta.amenity_id = a.id
			WHERE ta.theater_id = t.id
		) ta ON true
		WHERE ST_DWithin(t.location, ST_SetSRID(ST_MakePoint($3, $4), 4326), 20000)
		ORDER BY t.location <-> ST_SetSRID(ST_MakePoint($3, $4), 4326)
		LIMIT $5 OFFSET $6;
	`

	args := []any{movieID, date, long, lat, pagination.Limit(), pagination.Offset()}
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	theaters := make([]domain.Theater, 0, pagination.PageSize)
	var totalCount int

	for rows.Next() {
		var amenitiesJson, hallsJson json.RawMessage
		var theater domain.Theater

		if err := rows.Scan(
			&theater.ID,
			&theater.Name,
			&theater.Address,
			&theater.City,
			&theater.District,
			&theater.Distance,
			&amenitiesJson,
			&hallsJson,
			&totalCount,
		); err != nil {
			return nil, nil, err
		}

		if len(amenitiesJson) > 0 {
			if err := json.Unmarshal(amenitiesJson, &theater.Amenities); err != nil {
				return nil, nil, err
			}
		}

		if len(hallsJson) > 0 {
			if err := json.Unmarshal(hallsJson, &theater.Halls); err != nil {
				return nil, nil, err
			}
		}

		theaters = append(theaters, theater)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	metadata := domain.NewMetadata(totalCount, pagination.Page, pagination.PageSize)

	return theaters, metadata, nil
}
