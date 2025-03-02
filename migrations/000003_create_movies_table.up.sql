CREATE TABLE IF NOT EXISTS movies (
    id bigserial PRIMARY KEY,
    title text NOT NULL,
    description text NOT NULL,
    genres text[] NOT NULL,
    language text NOT NULL,
    release_date date NOT NULL,
    duration integer NOT NULL,
    poster_url text NOT NULL,
    director text NOT NULL,
    cast_members text[] NOT NULL,
    rating numeric(3, 1) CHECK (rating >= 0.0 AND rating <= 10.0),
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS movies_title_idx ON movies USING GIN (to_tsvector('english', title));

CREATE INDEX IF NOT EXISTS movies_desc_idx ON movies USING GIN (to_tsvector('english', description));