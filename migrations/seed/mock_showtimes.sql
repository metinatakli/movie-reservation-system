WITH showtime_slots AS (
    SELECT 
        show_date_offset, 
        show_time
    FROM generate_series(0, 6) AS show_date_offset
    CROSS JOIN LATERAL UNNEST(ARRAY[
        '10:00:00'::time, 
        '14:00:00'::time, 
        '18:00:00'::time, 
        '22:00:00'::time
    ]) AS show_time
)
INSERT INTO showtimes (movie_id, hall_id, start_time, base_price)
SELECT 
    m.id AS movie_id,
    h.id AS hall_id,
    (NOW()::date + s.show_date_offset + s.show_time::interval)::timestamptz AS start_time, 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM hall_amenities ha 
            JOIN amenities a ON ha.amenity_id = a.id 
            WHERE ha.hall_id = h.id 
            AND a.name IN ('IMAX', '4DX Experience')
        ) THEN 20.00
        WHEN EXISTS (
            SELECT 1 FROM hall_amenities ha 
            JOIN amenities a ON ha.amenity_id = a.id 
            WHERE ha.hall_id = h.id 
            AND a.name = 'Dolby Atmos Sound'
        ) THEN 14.99
        ELSE 12.00
    END AS base_price
FROM halls h
JOIN showtime_slots s ON TRUE
JOIN LATERAL (
    SELECT m.id 
    FROM movies m
    ORDER BY random() 
    LIMIT 1
) m ON TRUE
ON CONFLICT (hall_id, start_time) DO NOTHING;

