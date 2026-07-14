-- Restores migration 9's single shared trigger function. Note that doing so
-- reinstates the bug it was written to fix: inserting a comment will fail again,
-- because NEW.is_like cannot be planned against a comments row.

DROP TRIGGER IF EXISTS update_like_count ON likes;
DROP TRIGGER IF EXISTS update_comment_count ON comments;
DROP FUNCTION IF EXISTS update_like_counts();
DROP FUNCTION IF EXISTS update_comment_counts();

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
