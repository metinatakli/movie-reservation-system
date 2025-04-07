CREATE TYPE payment_status AS ENUM ('pending', 'canceled', 'completed', 'refunded');

CREATE TABLE payments (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id),
    stripe_checkout_session_id text UNIQUE NOT NULL,
    stripe_payment_intent_id  text UNIQUE,
    amount DECIMAL(8, 2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    payment_method text,
    status payment_status NOT NULL,
    error_message text,
    payment_date timestamp(0) with time zone,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone
);