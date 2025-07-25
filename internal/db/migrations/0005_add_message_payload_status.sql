ALTER TABLE messages
    ADD COLUMN payload JSONB,
    ADD COLUMN status TEXT DEFAULT 'received',
    ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT now();
CREATE INDEX IF NOT EXISTS messages_msg_id_idx ON messages(msg_id);
