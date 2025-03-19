DROP INDEX IF EXISTS showtimes_hall_movie_date_idx;
DROP TABLE IF EXISTS showtimes;

DROP INDEX IF EXISTS hall_amenities_hall_id_idx;
DROP TABLE IF EXISTS hall_amenities;

DROP INDEX IF EXISTS theater_amenities_theater_id_idx;
DROP TABLE IF EXISTS theater_amenities;

DROP TABLE IF EXISTS amenities;

DROP INDEX IF EXISTS halls_theater_id_idx;
DROP TABLE IF EXISTS halls;

DROP INDEX IF EXISTS theaters_location_idx;
DROP TABLE IF EXISTS theaters;