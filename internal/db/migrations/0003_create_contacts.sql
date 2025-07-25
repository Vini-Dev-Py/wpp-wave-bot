CREATE TABLE IF NOT EXISTS contacts (
    jid TEXT PRIMARY KEY,
    company_id TEXT NOT NULL,
    name TEXT,
    phone TEXT,
    last_seen TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
CREATE INDEX ON contacts(company_id);
