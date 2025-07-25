CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    company_id TEXT NOT NULL,
    msg_id TEXT NOT NULL,
    sender TEXT NOT NULL,
    receiver TEXT NOT NULL,
    type TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
CREATE INDEX ON messages(company_id);
