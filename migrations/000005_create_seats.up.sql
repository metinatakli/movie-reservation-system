CREATE TYPE seat_type AS ENUM ('Standard', 'VIP', 'Recliner', 'Accessible');

CREATE TABLE IF NOT EXISTS seats (
    id bigserial PRIMARY KEY,
    hall_id bigint NOT NULL REFERENCES halls ON DELETE CASCADE,
    seat_row integer NOT NULL,
    seat_col integer NOT NULL,
    seat_type seat_type NOT NULL,
    extra_price numeric(6,2) DEFAULT 0 NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    UNIQUE (hall_id, seat_row, seat_col)
);

CREATE INDEX IF NOT EXISTS seats_hall_id_idx ON seats(hall_id);