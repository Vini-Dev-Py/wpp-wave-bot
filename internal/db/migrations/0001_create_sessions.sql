CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    company_id TEXT NOT NULL UNIQUE,
    data BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
