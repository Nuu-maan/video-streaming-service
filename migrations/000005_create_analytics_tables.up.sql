CREATE TABLE IF NOT EXISTS video_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    quality VARCHAR(10),
    watch_duration INT DEFAULT 0,
    watch_percent FLOAT DEFAULT 0,
    device_type VARCHAR(20),
    country VARCHAR(2),
    source VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_video_views_video_id ON video_views(video_id);
CREATE INDEX idx_video_views_user_id ON video_views(user_id);
CREATE INDEX idx_video_views_created_at ON video_views(created_at DESC);
CREATE INDEX idx_video_views_country ON video_views(country);
CREATE INDEX idx_video_views_device_type ON video_views(device_type);
