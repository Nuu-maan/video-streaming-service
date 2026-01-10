DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'report_type') THEN
        CREATE TYPE report_type AS ENUM ('spam', 'harassment', 'hate_speech', 'violence', 'copyright', 'nudity', 'misinformation', 'other');
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'report_status') THEN
        CREATE TYPE report_status AS ENUM ('pending', 'reviewing', 'resolved', 'dismissed');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS content_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    video_id UUID REFERENCES videos(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    comment_id UUID,
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    report_type report_type NOT NULL,
    reason VARCHAR(100) NOT NULL,
    description TEXT,
    status report_status DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    action VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT report_target CHECK (
        (video_id IS NOT NULL) OR (user_id IS NOT NULL) OR (comment_id IS NOT NULL)
    )
);

CREATE INDEX idx_reports_status ON content_reports(status);
CREATE INDEX idx_reports_video_id ON content_reports(video_id);
CREATE INDEX idx_reports_user_id ON content_reports(user_id);
CREATE INDEX idx_reports_reporter_id ON content_reports(reporter_id);
CREATE INDEX idx_reports_created_at ON content_reports(created_at DESC);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id UUID,
    ip_address VARCHAR(45),
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_target ON audit_logs(target_type, target_id);
