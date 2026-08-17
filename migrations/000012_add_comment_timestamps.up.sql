-- Anchor a comment to a moment in the video it sits on.
--
-- Nullable on purpose: a comment about the video as a whole is not a comment at
-- second zero, and collapsing the two would make every general remark sort to
-- the start of the timeline. NULL means "no anchor", 0 means "the first frame".
ALTER TABLE comments ADD COLUMN video_timestamp INTEGER;

-- Seconds from the start. The upper bound is the video's duration, which lives
-- on another table and can be re-probed by a retranscode, so it is enforced in
-- the service rather than frozen into a constraint here.
ALTER TABLE comments ADD CONSTRAINT comments_video_timestamp_non_negative
    CHECK (video_timestamp IS NULL OR video_timestamp >= 0);

-- Serves the timestamp-ordered listing. Partial on the same predicates the
-- listing uses, so it stays small: anchored top-level comments only.
CREATE INDEX idx_comments_video_timestamp
    ON comments (video_id, video_timestamp)
    WHERE parent_id IS NULL AND video_timestamp IS NOT NULL AND deleted_at IS NULL;
