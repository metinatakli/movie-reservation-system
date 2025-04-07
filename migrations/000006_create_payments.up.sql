CREATE TABLE payments (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id),
    stripe_checkout_session_id text UNIQUE NOT NULL,
    amount DECIMAL(8, 2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status text NOT NULL CONSTRAINT payment_status CHECK (status IN ('pending', 'canceled', 'completed', 'refunded')),
    error_message text,
    payment_date timestamp(0) with time zone,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone
);