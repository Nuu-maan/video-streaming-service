package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orchids/video-streaming/internal/domain"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, report *domain.ContentReport) error {
	query := `
	INSERT INTO content_reports (id, video_id, user_id, comment_id, reporter_id, report_type, reason, description, status, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		report.ID, report.VideoID, report.UserID, report.CommentID, report.ReporterID,
		report.ReportType, report.Reason, report.Description, report.Status,
		report.CreatedAt, report.UpdatedAt,
	)
	return err
}

func (r *ReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ContentReport, error) {
	query := `
	SELECT id, video_id, user_id, comment_id, reporter_id, report_type, reason, description, 
		   status, reviewed_by, reviewed_at, action, created_at, updated_at
	FROM content_reports
	WHERE id = $1
	`

	report := &domain.ContentReport{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&report.ID, &report.VideoID, &report.UserID, &report.CommentID, &report.ReporterID,
		&report.ReportType, &report.Reason, &report.Description, &report.Status,
		&report.ReviewedBy, &report.ReviewedAt, &report.Action, &report.CreatedAt, &report.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrReportNotFound
		}
		return nil, err
	}

	return report, nil
}

func (r *ReportRepository) GetPending(ctx context.Context, limit, offset int) ([]*domain.ContentReport, error) {
	query := `
	SELECT id, video_id, user_id, comment_id, reporter_id, report_type, reason, description, 
		   status, reviewed_by, reviewed_at, action, created_at, updated_at
	FROM content_reports
	WHERE status = 'pending'
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*domain.ContentReport
	for rows.Next() {
		report := &domain.ContentReport{}
		err := rows.Scan(
			&report.ID, &report.VideoID, &report.UserID, &report.CommentID, &report.ReporterID,
			&report.ReportType, &report.Reason, &report.Description, &report.Status,
			&report.ReviewedBy, &report.ReviewedAt, &report.Action, &report.CreatedAt, &report.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func (r *ReportRepository) Update(ctx context.Context, report *domain.ContentReport) error {
	query := `
	UPDATE content_reports
	SET status = $1, reviewed_by = $2, reviewed_at = $3, action = $4, updated_at = $5
	WHERE id = $6
	`

	_, err := r.db.Exec(ctx, query,
		report.Status, report.ReviewedBy, report.ReviewedAt, report.Action, report.UpdatedAt, report.ID,
	)
	return err
}

func (r *ReportRepository) CountByStatus(ctx context.Context, status domain.ReportStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM content_reports WHERE status = $1`
	var count int64
	err := r.db.QueryRow(ctx, query, status).Scan(&count)
	return count, err
}
