CREATE TABLE IF NOT EXISTS reservations (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id),
    showtime_id bigint NOT NULL REFERENCES showtimes(id),
    payment_id bigint NOT NULL REFERENCES payments(id),
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone
);

CREATE TABLE IF NOT EXISTS reservation_seats (
    reservation_id bigint NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    showtime_id bigint NOT NULL REFERENCES showtimes(id),
    seat_id bigint NOT NULL REFERENCES seats(id),
    PRIMARY KEY (reservation_id, showtime_id, seat_id),
    CONSTRAINT unique_showtime_seat UNIQUE (showtime_id, seat_id)
);

CREATE INDEX IF NOT EXISTS reservation_seats_showtime_id_idx ON reservation_seats(showtime_id);