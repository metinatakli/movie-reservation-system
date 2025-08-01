INSERT INTO users  (id, first_name, last_name, email, password_hash, birth_date, gender, activated) VALUES 
(1, 'Test', 'User', 'test@user.com', decode('','hex'), '1970-01-01', 'M', true);

INSERT INTO payments (id, user_id, stripe_checkout_session_id, amount, status) VALUES
(1, 1, 'stripe', 189.99, 'completed');

INSERT INTO reservations (id, user_id, showtime_id, payment_id) VALUES 
(1, 1, 1, 1);

INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id) VALUES
(1, 1, 2);