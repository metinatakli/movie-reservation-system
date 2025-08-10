TRUNCATE TABLE 
    users,
    movies,
    theaters,
    halls,
    amenities,
    theater_amenities,
    hall_amenities,
    showtimes,
    seats,
    payments,
    reservations,
    reservation_seats
RESTART IDENTITY CASCADE;

INSERT INTO users (id, first_name, last_name, email, password_hash, birth_date, gender, activated) VALUES 
(1, 'Test', 'UserOne', 'user1@test.com', '$2b$12$w5kBl0DbdVJ0Vh8Mcu5jQ.2wlf.5V2TZcQ6eY09N6lQj1OGLKfUBC', '1970-01-01', 'M', true),
(2, 'Another', 'UserTwo', 'user2@test.com', '$2b$12$w5kBl0DbdVJ0Vh8Mcu5jQ.2wlf.5V2TZcQ6eY09N6lQj1OGLKfUBC', '1980-01-01', 'F', true);

INSERT INTO movies (id, title, description, genres, language, release_date, duration, poster_url, director, cast_members, rating) VALUES 
(1, 'The Go Story', 'A tale of concurrency and channels.', '{Action,Drama}', 'English', '2025-01-15', 125, 'https://example.com/poster-go.jpg', 'Rob Pike', '{"Ken Thompson","Robert Griesemer"}', 8.5);

INSERT INTO theaters (id, name, address, city, district, location, phone_number, email, website) VALUES 
(1, 'Grand Cinema', '123 Gopher Way', 'GoLand', 'Central', 'POINT(-74.0060 40.7128)', '555-1234', 'contact@grandcinema.dev', 'https://grandcinema.dev');

INSERT INTO halls (id, theater_id, name) VALUES 
(1, 1, 'Hall A');

INSERT INTO amenities (id, name, description) VALUES
(1, 'Cafe', 'Serves coffee and snacks.'),
(2, 'Parking', 'On-site parking available.'),
(3, 'IMAX', 'Large-format screen.'),
(4, 'Dolby Atmos', 'Immersive sound system.');

INSERT INTO theater_amenities (theater_id, amenity_id) VALUES (1, 1), (1, 2);
INSERT INTO hall_amenities (hall_id, amenity_id) VALUES (1, 3), (1, 4);

INSERT INTO showtimes (id, movie_id, hall_id, start_time, base_price) VALUES 
(1, 1, 1, '2095-05-10T20:00:00Z', 12.50);

INSERT INTO seats (id, hall_id, seat_row, seat_col, seat_type, extra_price) VALUES 
(10, 1, 3, 5, 'Standard', 0.00),
(11, 1, 3, 6, 'Standard', 0.00),
(12, 1, 4, 7, 'VIP', 5.00);

-- Reservation for User 1 (ID=1)
INSERT INTO payments (id, user_id, amount, status, stripe_checkout_session_id) VALUES 
(1, 1, 25.00, 'completed', 'cs_user1_res1');
INSERT INTO reservations (id, user_id, showtime_id, payment_id, created_at) VALUES 
(1, 1, 1, 1, '2095-04-20T10:00:00Z');
INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id) VALUES 
(1, 1, 10), 
(1, 1, 11);

-- Reservation for User 2 (ID=2)
INSERT INTO payments (id, user_id, amount, status, stripe_checkout_session_id) VALUES 
(2, 2, 17.50, 'completed', 'cs_user2_res2'); -- 12.50 base + 5.00 extra
INSERT INTO reservations (id, user_id, showtime_id, payment_id, created_at) VALUES 
(2, 2, 1, 2, '2095-04-21T11:00:00Z');
INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id) VALUES 
(2, 1, 12);