-- name: CreateVideo :one
INSERT INTO videos (
    id,
    title,
    description,
    filename,
    file_path,
    file_size,
    duration,
    status,
    mime_type,
    original_resolution,
    transcoding_progress,
    available_qualities
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetVideoByID :one
SELECT * FROM videos
WHERE id = $1;

-- name: ListVideos :many
SELECT * FROM videos
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateVideoStatus :exec
UPDATE videos
SET status = $2
WHERE id = $1;

-- name: UpdateVideoProgress :exec
UPDATE videos
SET transcoding_progress = $2
WHERE id = $1;

-- name: MarkVideoAsReady :exec
UPDATE videos
SET status = 'ready',
    available_qualities = $2,
    thumbnail_path = $3,
    transcoding_progress = 100,
    processed_at = NOW()
WHERE id = $1;

-- name: DeleteVideo :exec
DELETE FROM videos
WHERE id = $1;

-- name: GetVideosByStatus :many
SELECT * FROM videos
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SearchVideos :many
SELECT * FROM videos
WHERE to_tsvector('english', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateVideoDuration :exec
UPDATE videos
SET duration = $2
WHERE id = $1;

-- name: UpdateVideoResolution :exec
UPDATE videos
SET original_resolution = $2
WHERE id = $1;

-- name: MarkVideoAsFailed :exec
UPDATE videos
SET status = 'failed'
WHERE id = $1;
