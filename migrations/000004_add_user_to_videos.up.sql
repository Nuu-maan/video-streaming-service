CREATE TYPE video_visibility AS ENUM ('public', 'private', 'unlisted');

ALTER TABLE videos 
ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE,
ADD COLUMN visibility video_visibility DEFAULT 'public' NOT NULL;

CREATE INDEX idx_videos_user_id ON videos(user_id);
CREATE INDEX idx_videos_visibility ON videos(visibility);
CREATE INDEX idx_videos_user_visibility ON videos(user_id, visibility) WHERE user_id IS NOT NULL;
