-- Migration 9 pointed one trigger function at two different tables. Its first
-- branch reads
--
--     IF TG_TABLE_NAME = 'likes' AND NEW.is_like THEN
--
-- and PL/pgSQL resolves the field references in an expression when it plans the
-- expression, not when the guard in front of them happens to be true. Planning
-- NEW.is_like against a `comments` row — which has no is_like column — is a hard
-- error, so EVERY insert into comments failed with
--
--     record "new" has no field "is_like"
--
-- and no comment could ever be written. The same trap sits in the UPDATE and
-- DELETE branches via OLD.is_like.
--
-- One function per table removes the whole class of bug: a field reference can
-- no longer be compiled against a record type that lacks it.

DROP TRIGGER IF EXISTS update_like_count ON likes;
DROP TRIGGER IF EXISTS update_comment_count ON comments;
DROP FUNCTION IF EXISTS update_video_counts();

CREATE FUNCTION update_like_counts() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.is_like THEN
            UPDATE videos SET like_count = like_count + 1 WHERE id = NEW.video_id;
        END IF;
    ELSIF TG_OP = 'UPDATE' THEN
        -- A like flipping to a dislike (or back) is an UPDATE, not a delete and
        -- re-insert, because of UNIQUE(user_id, video_id).
        IF OLD.is_like IS DISTINCT FROM NEW.is_like THEN
            IF NEW.is_like THEN
                UPDATE videos SET like_count = like_count + 1 WHERE id = NEW.video_id;
            ELSE
                UPDATE videos SET like_count = like_count - 1 WHERE id = NEW.video_id;
            END IF;
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.is_like THEN
            UPDATE videos SET like_count = like_count - 1 WHERE id = OLD.video_id;
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION update_comment_counts() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.deleted_at IS NULL THEN
            UPDATE videos SET comment_count = comment_count + 1 WHERE id = NEW.video_id;
            IF NEW.parent_id IS NOT NULL THEN
                UPDATE comments SET reply_count = reply_count + 1 WHERE id = NEW.parent_id;
            END IF;
        END IF;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Comments are soft-deleted, so the count moves on a deleted_at
        -- transition rather than on a DELETE.
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
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.deleted_at IS NULL THEN
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
    FOR EACH ROW EXECUTE FUNCTION update_like_counts();

CREATE TRIGGER update_comment_count AFTER INSERT OR UPDATE OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_comment_counts();

-- Any counter that drifted while the trigger was broken. comment_count could
-- only ever be 0 (no comment could be inserted at all), but a like that was
-- inserted before this migration counted correctly, so recompute rather than
-- assume.
UPDATE videos v SET
    like_count    = (SELECT count(*) FROM likes l    WHERE l.video_id = v.id AND l.is_like),
    comment_count = (SELECT count(*) FROM comments c WHERE c.video_id = v.id AND c.deleted_at IS NULL);

UPDATE comments p SET
    reply_count = (SELECT count(*) FROM comments r WHERE r.parent_id = p.id AND r.deleted_at IS NULL);
