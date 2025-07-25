CREATE TABLE IF NOT EXISTS groups (
    jid TEXT PRIMARY KEY,
    company_id TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
CREATE INDEX IF NOT EXISTS groups_company_idx ON groups(company_id);
