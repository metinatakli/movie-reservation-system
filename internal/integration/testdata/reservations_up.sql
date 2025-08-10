INSERT INTO payments (id, user_id, amount, status) VALUES
(1, 1, 15.50, 'completed'),
(2, 1, 15.50, 'completed'),
(3, 1, 15.50, 'completed'),
(4, 1, 15.50, 'completed'),
(5, 1, 15.50, 'completed'),
(6, 1, 15.50, 'completed'),
(7, 1, 15.50, 'completed');

INSERT INTO reservations (id, user_id, showtime_id, payment_id, created_at) VALUES
(1, 1, 1, 1, '2095-02-01 10:00:07Z'),
(2, 1, 2, 2, '2095-02-01 10:00:06Z'),
(3, 1, 3, 3, '2095-02-01 10:00:05Z'),
(4, 1, 4, 4, '2095-02-01 10:00:04Z'),
(5, 1, 5, 5, '2095-02-01 10:00:03Z'),
(6, 1, 6, 6, '2095-02-01 10:00:02Z'),
(7, 1, 7, 7, '2095-02-01 10:00:01Z');
