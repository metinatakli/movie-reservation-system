CREATE EXTENSION IF NOT EXISTS "citext";    -- For case-insensitive email addresses

CREATE TYPE gender_type AS ENUM ('M', 'F', 'OTHER');

CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    first_name text NOT NULL,
    last_name text NOT NULL,
    email citext NOT NULL,
    password_hash bytea NOT NULL,
    birth_date date NOT NULL,
    gender gender_type NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone,
    activated boolean NOT NULL DEFAULT FALSE,
    is_active boolean NOT NULL DEFAULT TRUE,
    version integer NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX users_email_active_idx ON users (email) WHERE is_active = TRUE;