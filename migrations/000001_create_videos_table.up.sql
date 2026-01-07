CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE video_status AS ENUM ('uploading', 'processing', 'ready', 'failed');

CREATE TABLE IF NOT EXISTS videos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    filename VARCHAR(500) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    duration INTEGER DEFAULT 0,
    status video_status NOT NULL DEFAULT 'uploading',
    mime_type VARCHAR(100) NOT NULL,
    original_resolution VARCHAR(50),
    thumbnail_path VARCHAR(1000),
    transcoding_progress INTEGER DEFAULT 0 CHECK (transcoding_progress >= 0 AND transcoding_progress <= 100),
    available_qualities TEXT[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_videos_status ON videos(status);
CREATE INDEX idx_videos_created_at ON videos(created_at DESC);
CREATE INDEX idx_videos_title_search ON videos USING gin(to_tsvector('english', title || ' ' || COALESCE(description, '')));

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_videos_updated_at
    BEFORE UPDATE ON videos
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
