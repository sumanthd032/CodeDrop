CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS drops (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    max_downloads INT NOT NULL,
    current_downloads INT DEFAULT 0,
    file_name TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type TEXT,
    encryption_salt TEXT NOT NULL, -- Used to derive the key on the client side
    is_deleted BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS chunks (
    id SERIAL PRIMARY KEY,
    drop_id UUID REFERENCES drops(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    chunk_hash TEXT NOT NULL, -- SHA-256 hash to verify integrity
    size INT NOT NULL,
    UNIQUE(drop_id, chunk_index)
);

CREATE INDEX idx_drops_expires_at ON drops(expires_at);