-- Create subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscriber_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notify_uploads BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(subscriber_id, creator_id),
    CHECK (subscriber_id != creator_id)
);

CREATE INDEX idx_subscriptions_subscriber ON subscriptions(subscriber_id);
CREATE INDEX idx_subscriptions_creator ON subscriptions(creator_id);
CREATE INDEX idx_subscriptions_created_at ON subscriptions(created_at DESC);

-- Create likes table
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    is_like BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_id, video_id)
);

CREATE INDEX idx_likes_video ON likes(video_id);
CREATE INDEX idx_likes_user ON likes(user_id);
CREATE INDEX idx_likes_user_video ON likes(user_id, video_id);

-- Create comments table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    like_count BIGINT DEFAULT 0,
    reply_count BIGINT DEFAULT 0,
    pinned BOOLEAN DEFAULT false,
    edited_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    CHECK (char_length(content) <= 10000),
    CHECK (char_length(content) > 0)
);

CREATE INDEX idx_comments_video ON comments(video_id, created_at DESC);
CREATE INDEX idx_comments_user ON comments(user_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);
CREATE INDEX idx_comments_deleted ON comments(deleted_at) WHERE deleted_at IS NULL;

-- Create playlists table
CREATE TABLE playlists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(20) DEFAULT 'public' CHECK (visibility IN ('public', 'private', 'unlisted')),
    video_count BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_playlists_user ON playlists(user_id);
CREATE INDEX idx_playlists_visibility ON playlists(visibility);

-- Create playlist_videos table
CREATE TABLE playlist_videos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    playlist_id UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(playlist_id, video_id),
    UNIQUE(playlist_id, position)
);

CREATE INDEX idx_playlist_videos_playlist ON playlist_videos(playlist_id, position);
CREATE INDEX idx_playlist_videos_video ON playlist_videos(video_id);

-- Create watch_history table
CREATE TABLE watch_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    watched_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    watch_duration INTEGER NOT NULL,
    completed BOOLEAN DEFAULT false,
    last_position INTEGER DEFAULT 0,
    
    UNIQUE(user_id, video_id)
);

CREATE INDEX idx_watch_history_user ON watch_history(user_id, watched_at DESC);
CREATE INDEX idx_watch_history_video ON watch_history(video_id);

-- Create watch_later table
CREATE TABLE watch_later (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_id, video_id)
);

CREATE INDEX idx_watch_later_user ON watch_later(user_id, added_at DESC);
CREATE INDEX idx_watch_later_video ON watch_later(video_id);

-- Create notification_type enum
CREATE TYPE notification_type AS ENUM (
    'new_video', 'comment', 'reply', 'like', 'subscriber', 'mention'
);

-- Create notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type notification_type NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    action_url TEXT,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    video_id UUID REFERENCES videos(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    read BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user ON notifications(user_id, read, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id) WHERE read = false;

-- Add subscriber_count to users table
ALTER TABLE users
ADD COLUMN subscriber_count BIGINT DEFAULT 0;

CREATE INDEX idx_users_subscriber_count ON users(subscriber_count DESC);

-- Triggers to automatically update counts

-- Update video counts (likes, comments)
CREATE OR REPLACE FUNCTION update_video_counts() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF TG_TABLE_NAME = 'likes' AND NEW.is_like THEN
            UPDATE videos SET like_count = like_count + 1 WHERE id = NEW.video_id;
        ELSIF TG_TABLE_NAME = 'comments' AND NEW.deleted_at IS NULL THEN
            UPDATE videos SET comment_count = comment_count + 1 WHERE id = NEW.video_id;
            IF NEW.parent_id IS NOT NULL THEN
                UPDATE comments SET reply_count = reply_count + 1 WHERE id = NEW.parent_id;
            END IF;
        END IF;
    ELSIF TG_OP = 'UPDATE' THEN
        IF TG_TABLE_NAME = 'likes' THEN
            IF OLD.is_like != NEW.is_like THEN
                IF NEW.is_like THEN
                    UPDATE videos SET like_count = like_count + 1 WHERE id = NEW.video_id;
                ELSE
                    UPDATE videos SET like_count = like_count - 1 WHERE id = NEW.video_id;
                END IF;
            END IF;
        ELSIF TG_TABLE_NAME = 'comments' THEN
            IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
                UPDATE videos SET comment_count = comment_count - 1 WHERE id = NEW.video_id;
                IF NEW.parent_id IS NOT NULL THEN
                    UPDATE comments SET reply_count = reply_count - 1 WHERE id = NEW.parent_id;
                END IF;
            ELSIF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL THEN
                UPDATE videos SET comment_count = comment_count + 1 WHERE id = NEW.video_id;
                IF NEW.parent_id IS NOT NULL THEN
                    UPDATE comments SET reply_count = reply_count + 1 WHERE id = NEW.parent_id;
                END IF;
            END IF;
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        IF TG_TABLE_NAME = 'likes' AND OLD.is_like THEN
            UPDATE videos SET like_count = like_count - 1 WHERE id = OLD.video_id;
        ELSIF TG_TABLE_NAME = 'comments' AND OLD.deleted_at IS NULL THEN
            UPDATE videos SET comment_count = comment_count - 1 WHERE id = OLD.video_id;
            IF OLD.parent_id IS NOT NULL THEN
                UPDATE comments SET reply_count = reply_count - 1 WHERE id = OLD.parent_id;
            END IF;
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_like_count AFTER INSERT OR UPDATE OR DELETE ON likes
    FOR EACH ROW EXECUTE FUNCTION update_video_counts();

CREATE TRIGGER update_comment_count AFTER INSERT OR UPDATE OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_video_counts();

-- Update subscriber count
CREATE OR REPLACE FUNCTION update_subscriber_count() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE users SET subscriber_count = subscriber_count + 1 WHERE id = NEW.creator_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE users SET subscriber_count = subscriber_count - 1 WHERE id = OLD.creator_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_user_subscribers AFTER INSERT OR DELETE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_subscriber_count();

-- Update playlist video count
CREATE OR REPLACE FUNCTION update_playlist_count() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE playlists SET video_count = video_count + 1 WHERE id = NEW.playlist_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE playlists SET video_count = video_count - 1 WHERE id = OLD.playlist_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_playlist_video_count AFTER INSERT OR DELETE ON playlist_videos
    FOR EACH ROW EXECUTE FUNCTION update_playlist_count();

-- Update comment updated_at timestamp
CREATE OR REPLACE FUNCTION update_comment_timestamp() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_comment_updated_at BEFORE UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_comment_timestamp();

-- Update playlist updated_at timestamp
CREATE OR REPLACE FUNCTION update_playlist_timestamp() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_playlist_updated_at BEFORE UPDATE ON playlists
    FOR EACH ROW EXECUTE FUNCTION update_playlist_timestamp();
