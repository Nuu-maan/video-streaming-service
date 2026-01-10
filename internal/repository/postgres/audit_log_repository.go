package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orchids/video-streaming/internal/domain"
)

type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO audit_logs (id, user_id, action, target_type, target_id, ip_address, user_agent, details, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.Exec(ctx, query,
		log.ID, log.UserID, log.Action, log.TargetType, log.TargetID,
		log.IPAddress, log.UserAgent, detailsJSON, log.CreatedAt,
	)
	return err
}

func (r *AuditLogRepository) GetRecent(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
	SELECT id, user_id, action, target_type, target_id, ip_address, user_agent, details, created_at
	FROM audit_logs
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		log := &domain.AuditLog{}
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.TargetType, &log.TargetID,
			&log.IPAddress, &log.UserAgent, &detailsJSON, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &log.Details)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func (r *AuditLogRepository) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
	SELECT id, user_id, action, target_type, target_id, ip_address, user_agent, details, created_at
	FROM audit_logs
	WHERE user_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		log := &domain.AuditLog{}
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.TargetType, &log.TargetID,
			&log.IPAddress, &log.UserAgent, &detailsJSON, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &log.Details)
		}

		logs = append(logs, log)
	}

	return logs, nil
}
