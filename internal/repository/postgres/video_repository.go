package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orchids/video-streaming/internal/domain"
)

type PostgresVideoRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresVideoRepository(pool *pgxpool.Pool) *PostgresVideoRepository {
	return &PostgresVideoRepository{
		pool: pool,
	}
}

func (r *PostgresVideoRepository) Create(ctx context.Context, video *domain.Video) error {
	query := `
		INSERT INTO videos (
			id, title, description, filename, file_path, file_size, mime_type,
			duration, original_resolution, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := r.pool.Exec(ctx, query,
		video.ID,
		video.Title,
		video.Description,
		video.Filename,
		video.FilePath,
		video.FileSize,
		video.MimeType,
		video.Duration,
		video.OriginalResolution,
		video.Status,
		video.CreatedAt,
		video.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create video: %w", err)
	}

	return nil
}

func (r *PostgresVideoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Video, error) {
	query := `
		SELECT id, title, description, filename, file_path, file_size, mime_type,
			   duration, original_resolution, thumbnail_path, status,
			   transcoding_progress, available_qualities, hls_master_path, hls_ready, streaming_protocol,
			   created_at, updated_at, processed_at
		FROM videos
		WHERE id = $1
	`

	var video domain.Video
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&video.ID,
		&video.Title,
		&video.Description,
		&video.Filename,
		&video.FilePath,
		&video.FileSize,
		&video.MimeType,
		&video.Duration,
		&video.OriginalResolution,
		&video.ThumbnailPath,
		&video.Status,
		&video.TranscodingProgress,
		&video.AvailableQualities,
		&video.HLSMasterPath,
		&video.HLSReady,
		&video.StreamingProtocol,
		&video.CreatedAt,
		&video.UpdatedAt,
		&video.ProcessedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVideoNotFound
		}
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	return &video, nil
}

func (r *PostgresVideoRepository) List(ctx context.Context, limit, offset int) ([]*domain.Video, error) {
	query := `
		SELECT id, title, description, filename, file_path, file_size, mime_type,
			   duration, original_resolution, thumbnail_path, status,
			   transcoding_progress, available_qualities, hls_master_path, hls_ready, streaming_protocol,
			   created_at, updated_at, processed_at
		FROM videos
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list videos: %w", err)
	}
	defer rows.Close()

	var videos []*domain.Video
	for rows.Next() {
		var video domain.Video
		err := rows.Scan(
			&video.ID,
			&video.Title,
			&video.Description,
			&video.Filename,
			&video.FilePath,
			&video.FileSize,
			&video.MimeType,
			&video.Duration,
			&video.OriginalResolution,
			&video.ThumbnailPath,
			&video.Status,
			&video.TranscodingProgress,
			&video.AvailableQualities,
			&video.HLSMasterPath,
			&video.HLSReady,
			&video.StreamingProtocol,
			&video.CreatedAt,
			&video.UpdatedAt,
			&video.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, &video)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating videos: %w", err)
	}

	return videos, nil
}

func (r *PostgresVideoRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.VideoStatus) error {
	query := `
		UPDATE videos
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}

func (r *PostgresVideoRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress int) error {
	query := `
		UPDATE videos
		SET transcoding_progress = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, progress, id)
	if err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}

func (r *PostgresVideoRepository) MarkAsReady(ctx context.Context, id uuid.UUID, qualities []string, thumbnailPath string) error {
	query := `
		UPDATE videos
		SET status = $1,
		    available_qualities = $2,
		    thumbnail_path = $3,
		    processed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $4
	`

	result, err := r.pool.Exec(ctx, query, domain.VideoStatusReady, qualities, thumbnailPath, id)
	if err != nil {
		return fmt.Errorf("failed to mark as ready: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}

func (r *PostgresVideoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM videos WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}

func (r *PostgresVideoRepository) GetByStatus(ctx context.Context, status domain.VideoStatus, limit, offset int) ([]*domain.Video, error) {
	query := `
		SELECT id, title, description, filename, file_path, file_size, mime_type,
			   duration, original_resolution, thumbnail_path, status,
			   transcoding_progress, available_qualities, created_at, updated_at, processed_at
		FROM videos
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos by status: %w", err)
	}
	defer rows.Close()

	var videos []*domain.Video
	for rows.Next() {
		var video domain.Video
		err := rows.Scan(
			&video.ID,
			&video.Title,
			&video.Description,
			&video.Filename,
			&video.FilePath,
			&video.FileSize,
			&video.MimeType,
			&video.Duration,
			&video.OriginalResolution,
			&video.ThumbnailPath,
			&video.Status,
			&video.TranscodingProgress,
			&video.AvailableQualities,
			&video.CreatedAt,
			&video.UpdatedAt,
			&video.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, &video)
	}

	return videos, nil
}

func (r *PostgresVideoRepository) Search(ctx context.Context, query string, limit, offset int) ([]*domain.Video, error) {
	sqlQuery := `
		SELECT id, title, description, filename, file_path, file_size, mime_type,
			   duration, original_resolution, thumbnail_path, status,
			   transcoding_progress, available_qualities, created_at, updated_at, processed_at
		FROM videos
		WHERE title ILIKE '%' || $1 || '%' OR COALESCE(description, '') ILIKE '%' || $1 || '%'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, sqlQuery, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}
	defer rows.Close()

	var videos []*domain.Video
	for rows.Next() {
		var video domain.Video
		err := rows.Scan(
			&video.ID,
			&video.Title,
			&video.Description,
			&video.Filename,
			&video.FilePath,
			&video.FileSize,
			&video.MimeType,
			&video.Duration,
			&video.OriginalResolution,
			&video.ThumbnailPath,
			&video.Status,
			&video.TranscodingProgress,
			&video.AvailableQualities,
			&video.CreatedAt,
			&video.UpdatedAt,
			&video.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, &video)
	}

	return videos, nil
}

func (r *PostgresVideoRepository) UpdateDuration(ctx context.Context, id uuid.UUID, duration int) error {
	query := `
		UPDATE videos
		SET duration = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, duration, id)
	if err != nil {
		return fmt.Errorf("failed to update duration: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}

func (r *PostgresVideoRepository) UpdateResolution(ctx context.Context, id uuid.UUID, resolution string) error {
	query := `
		UPDATE videos
		SET original_resolution = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, resolution, id)
	if err != nil {
		return fmt.Errorf("failed to update resolution: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}

func (r *PostgresVideoRepository) MarkAsFailed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE videos
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, domain.VideoStatusFailed, id)
	if err != nil {
		return fmt.Errorf("failed to mark as failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}

func (r *PostgresVideoRepository) UpdateHLSInfo(ctx context.Context, id uuid.UUID, hlsMasterPath string, hlsReady bool) error {
	query := `
		UPDATE videos
		SET hls_master_path = $1, hls_ready = $2, streaming_protocol = $3, updated_at = NOW()
		WHERE id = $4
	`

	result, err := r.pool.Exec(ctx, query, hlsMasterPath, hlsReady, "hls", id)
	if err != nil {
		return fmt.Errorf("failed to update HLS info: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}

	return nil
}
