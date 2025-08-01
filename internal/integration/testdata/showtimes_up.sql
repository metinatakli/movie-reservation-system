INSERT INTO theaters (id, name, address, city, district, location, phone_number, email, website) VALUES
(1, 'Test Theater 1', '123 Main St', 'Test City', 'Central', 'POINT(30.0000 40.0000)', '555-1111', 'theater1@example.com', 'http://theater1.com'),
(2, 'Test Theater 2', '456 Side St', 'Test City', 'North', 'POINT(30.0000 40.0000)', '555-2222', 'theater2@example.com', 'http://theater2.com');

INSERT INTO halls (id, theater_id, name) VALUES
(1, 1, 'Hall 1A'),
(2, 1, 'Hall 1B'),
(3, 2, 'Hall 2A'),
(4, 2, 'Hall 2B');

INSERT INTO amenities (id, name, description) VALUES
(1, 'IMAX', 'Large-format screen'),
(2, 'Dolby Atmos', 'Immersive sound system');

INSERT INTO hall_amenities (hall_id, amenity_id) VALUES
(1, 1),
(2, 2),
(3, 1),
(4, 2);

INSERT INTO showtimes (id, movie_id, hall_id, start_time, base_price) VALUES
(1, 1, 1, '2095-01-01T10:00:00+00:00', 10.0),
(2, 1, 1, '2095-01-01T14:00:00+00:00', 12.0),
(3, 1, 2, '2095-01-01T10:00:00+00:00', 11.0),
(4, 1, 2, '2095-01-01T14:00:00+00:00', 13.0),
(5, 1, 3, '2095-01-01T10:00:00+00:00', 10.5),
(6, 1, 3, '2095-01-01T14:00:00+00:00', 12.5),
(7, 1, 4, '2095-01-01T10:00:00+00:00', 11.5),
(8, 1, 4, '2095-01-01T14:00:00+00:00', 13.5);
