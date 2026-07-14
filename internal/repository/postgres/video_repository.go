package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
)

// videoColumns is the single source of truth for the SELECT list. The three
// previous listing queries each hand-wrote their own column list, and two of
// them silently omitted the HLS columns, so a video fetched by search reported
// HLSReady=false regardless of its real state.
//
// The discovery and engagement columns are COALESCEd because migrations 8 and 9
// added them as nullable: a row written before those migrations has NULL there,
// not 0, and scanning a NULL into an int64 fails.
const videoColumns = `
	id, user_id, title, description, filename, file_path, file_size, mime_type,
	duration, original_resolution, thumbnail_path, status, visibility,
	transcoding_progress, available_qualities, hls_master_path, hls_ready,
	streaming_protocol,
	COALESCE(category, ''), tags, COALESCE(language, ''),
	COALESCE(view_count, 0), COALESCE(like_count, 0), COALESCE(comment_count, 0),
	created_at, updated_at, processed_at`

// PostgresVideoRepository is the PostgreSQL implementation of
// repository.VideoRepository.
type PostgresVideoRepository struct {
	pool *pgxpool.Pool
}

// Compile-time proof that the implementation satisfies the interface. Without
// this, a drifting method set only surfaces at the call site.
var _ repository.VideoRepository = (*PostgresVideoRepository)(nil)

func NewPostgresVideoRepository(pool *pgxpool.Pool) *PostgresVideoRepository {
	return &PostgresVideoRepository{pool: pool}
}

// scanner is satisfied by both pgx.Row and pgx.Rows, so one scan helper serves
// single-row and multi-row queries alike.
type scanner interface {
	Scan(dest ...any) error
}

// videoScanDest returns the scan destinations for videoColumns, in order.
//
// It is the counterpart to videoColumns and the two must be edited together, so
// they live side by side. Every query that selects videoColumns scans through
// here — including listings elsewhere in the package that join videos to a
// membership table and append their own destinations (see scanVideoWith). Those
// used to keep a second, hand-copied destination list, which silently drifted
// out of sync the moment a column was added and failed at runtime with
// "number of field descriptions must equal number of destinations".
func videoScanDest(v *domain.Video) []any {
	return []any{
		&v.ID,
		&v.UserID,
		&v.Title,
		&v.Description,
		&v.Filename,
		&v.FilePath,
		&v.FileSize,
		&v.MimeType,
		&v.Duration,
		&v.OriginalResolution,
		&v.ThumbnailPath,
		&v.Status,
		&v.Visibility,
		&v.TranscodingProgress,
		&v.AvailableQualities,
		&v.HLSMasterPath,
		&v.HLSReady,
		&v.StreamingProtocol,
		&v.Category,
		&v.Tags,
		&v.Language,
		&v.ViewCount,
		&v.LikeCount,
		&v.CommentCount,
		&v.CreatedAt,
		&v.UpdatedAt,
		&v.ProcessedAt,
	}
}

// scanVideo reads one row in videoColumns order.
func scanVideo(row scanner) (*domain.Video, error) {
	var video domain.Video
	if err := row.Scan(videoScanDest(&video)...); err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *PostgresVideoRepository) Create(ctx context.Context, video *domain.Video) error {
	const query = `
		INSERT INTO videos (
			id, user_id, title, description, filename, file_path, file_size,
			mime_type, duration, original_resolution, status, visibility,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		video.ID,
		video.UserID,
		video.Title,
		video.Description,
		video.Filename,
		video.FilePath,
		video.FileSize,
		video.MimeType,
		video.Duration,
		video.OriginalResolution,
		video.Status,
		video.Visibility,
		video.CreatedAt,
		video.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating video: %w", err)
	}
	return nil
}

func (r *PostgresVideoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Video, error) {
	// The space before FROM is load-bearing: videoColumns has no trailing
	// whitespace, so without it the last column and FROM fuse into
	// "processed_atFROM", which Postgres reads as a column alias — leaving the
	// statement with no FROM clause at all.
	query := `SELECT` + videoColumns + ` FROM videos WHERE id = $1`

	video, err := scanVideo(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVideoNotFound
		}
		return nil, fmt.Errorf("getting video %s: %w", id, err)
	}
	return video, nil
}

// buildFilter renders a VideoFilter into a WHERE clause and its arguments.
// Arguments are always bound as placeholders; no filter value is ever
// interpolated into the SQL text.
func buildFilter(filter repository.VideoFilter) (where string, args []any) {
	var conditions []string

	if filter.Status != nil {
		args = append(args, *filter.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.OwnerID != nil {
		args = append(args, *filter.OwnerID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if filter.Visibility != nil {
		args = append(args, *filter.Visibility)
		conditions = append(conditions, fmt.Sprintf("visibility = $%d", len(args)))
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		// Uses the search_vector GIN index added in migration 8. The old query
		// was an unanchored ILIKE '%...%', which cannot use an index and forced
		// a sequential scan of the table on every search.
		args = append(args, search)
		conditions = append(conditions, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (r *PostgresVideoRepository) List(ctx context.Context, filter repository.VideoFilter, page repository.Page) ([]*domain.Video, error) {
	where, args := buildFilter(filter)

	query := fmt.Sprintf(
		`SELECT %s FROM videos%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		videoColumns, where, len(args)+1, len(args)+2,
	)
	args = append(args, page.Limit, page.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing videos: %w", err)
	}
	defer rows.Close()

	videos := make([]*domain.Video, 0, page.Limit)
	for rows.Next() {
		video, err := scanVideo(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning video: %w", err)
		}
		videos = append(videos, video)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating videos: %w", err)
	}
	return videos, nil
}

func (r *PostgresVideoRepository) Count(ctx context.Context, filter repository.VideoFilter) (int, error) {
	where, args := buildFilter(filter)

	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM videos`+where, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting videos: %w", err)
	}
	return count, nil
}

func (r *PostgresVideoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.exec(ctx, `DELETE FROM videos WHERE id = $1`, id)
}

func (r *PostgresVideoRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.VideoStatus) error {
	return r.exec(ctx, `UPDATE videos SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
}

func (r *PostgresVideoRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress int) error {
	return r.exec(ctx, `UPDATE videos SET transcoding_progress = $2, updated_at = NOW() WHERE id = $1`, id, progress)
}

func (r *PostgresVideoRepository) UpdateDuration(ctx context.Context, id uuid.UUID, duration int) error {
	return r.exec(ctx, `UPDATE videos SET duration = $2, updated_at = NOW() WHERE id = $1`, id, duration)
}

func (r *PostgresVideoRepository) UpdateResolution(ctx context.Context, id uuid.UUID, resolution string) error {
	return r.exec(ctx, `UPDATE videos SET original_resolution = $2, updated_at = NOW() WHERE id = $1`, id, resolution)
}

func (r *PostgresVideoRepository) UpdateHLSInfo(ctx context.Context, id uuid.UUID, hlsMasterPath string, hlsReady bool) error {
	return r.exec(ctx,
		`UPDATE videos
		 SET hls_master_path = $2, hls_ready = $3, streaming_protocol = 'hls', updated_at = NOW()
		 WHERE id = $1`,
		id, hlsMasterPath, hlsReady,
	)
}

func (r *PostgresVideoRepository) MarkAsReady(ctx context.Context, id uuid.UUID, qualities []string, thumbnailPath string) error {
	return r.exec(ctx,
		`UPDATE videos
		 SET status = $2, available_qualities = $3, thumbnail_path = $4,
		     transcoding_progress = 100, processed_at = NOW(), updated_at = NOW()
		 WHERE id = $1`,
		id, domain.VideoStatusReady, qualities, thumbnailPath,
	)
}

func (r *PostgresVideoRepository) MarkAsFailed(ctx context.Context, id uuid.UUID) error {
	return r.exec(ctx, `UPDATE videos SET status = $2, updated_at = NOW() WHERE id = $1`, id, domain.VideoStatusFailed)
}

// exec runs a statement whose first argument is the video ID and reports
// ErrVideoNotFound when it matches no row. Every update method shared this
// eight-line body verbatim.
func (r *PostgresVideoRepository) exec(ctx context.Context, query string, id uuid.UUID, args ...any) error {
	tag, err := r.pool.Exec(ctx, query, append([]any{id}, args...)...)
	if err != nil {
		return fmt.Errorf("updating video %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVideoNotFound
	}
	return nil
}
