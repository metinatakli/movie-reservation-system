-- For geospatial queries

CREATE EXTENSION IF NOT EXISTS postgis;

-- Theaters

CREATE TABLE IF NOT EXISTS theaters (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    address text NOT NULL,
    city text NOT NULL,
    district text NOT NULL,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    phone_number text NOT NULL,
    email text NOT NULL,
    website text NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS theaters_location_idx ON theaters USING GIST(location);

-- Halls

CREATE TABLE IF NOT EXISTS halls (
    id bigserial PRIMARY KEY,
    theater_id bigint NOT NULL REFERENCES theaters ON DELETE CASCADE, 
    name text NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS halls_theater_id_idx ON halls (theater_id);

-- Amenities

CREATE TABLE IF NOT EXISTS amenities (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS theater_amenities (
    theater_id bigint NOT NULL REFERENCES theaters ON DELETE CASCADE,
    amenity_id bigint NOT NULL REFERENCES amenities ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (theater_id, amenity_id)
);

CREATE INDEX IF NOT EXISTS theater_amenities_theater_id_idx ON theater_amenities (theater_id);

CREATE TABLE IF NOT EXISTS hall_amenities (
    hall_id bigint NOT NULL REFERENCES halls ON DELETE CASCADE,
    amenity_id bigint NOT NULL REFERENCES amenities ON DELETE CASCADE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (hall_id, amenity_id)
);

CREATE INDEX IF NOT EXISTS hall_amenities_hall_id_idx ON hall_amenities (hall_id);

-- Showtimes

CREATE TABLE IF NOT EXISTS showtimes (
    id bigserial PRIMARY KEY,
    movie_id bigint NOT NULL REFERENCES movies ON DELETE CASCADE,
    hall_id bigint NOT NULL REFERENCES halls ON DELETE CASCADE,
    start_time timestamp(0) with time zone NOT NULL,
    base_price numeric(6, 2) NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    UNIQUE (hall_id, start_time)
);

CREATE INDEX IF NOT EXISTS showtimes_hall_movie_idx ON showtimes (hall_id, movie_id);
