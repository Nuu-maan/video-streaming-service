package service

import (
	"context"
	"fmt"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/google/uuid"
)

// AuditLogRepository is the slice of the audit log store this service needs,
// stated in the method names *postgres.AuditLogRepository actually has. The
// repository deliberately exposes fixed queries rather than a generic filter
// map: filtering is resolved here, in GetLogs.
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	GetRecent(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error)
	GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)
	Count(ctx context.Context) (int64, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
}

type AuditService struct {
	repo AuditLogRepository
}

func NewAuditService(repo AuditLogRepository) *AuditService {
	return &AuditService{
		repo: repo,
	}
}

func (s *AuditService) Log(ctx context.Context, action, targetType string, targetID *uuid.UUID, details map[string]interface{}) error {
	userID := getUserIDFromContext(ctx)
	ipAddress := getIPFromContext(ctx)
	userAgent := getUserAgentFromContext(ctx)

	log := domain.NewAuditLog(userID, action, targetType, targetID, ipAddress, userAgent, details)

	go func() {
		if err := s.repo.Create(context.Background(), log); err != nil {
		}
	}()

	return nil
}

// GetLogs returns one page of audit logs plus the total matching the filters.
//
// The only filter the audit log schema can serve is user_id (audit_logs is
// indexed on it); an unrecognised or empty filter set lists the most recent logs
// across all users.
func (s *AuditService) GetLogs(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.AuditLog, int64, error) {
	if userID, ok := auditUserFilter(filters); ok {
		logs, err := s.repo.GetByUser(ctx, userID, limit, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("listing audit logs for user %s: %w", userID, err)
		}

		total, err := s.repo.CountByUser(ctx, userID)
		if err != nil {
			return nil, 0, fmt.Errorf("counting audit logs for user %s: %w", userID, err)
		}

		return logs, total, nil
	}

	logs, err := s.repo.GetRecent(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing audit logs: %w", err)
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("counting audit logs: %w", err)
	}

	return logs, total, nil
}

// auditUserFilter extracts a user_id filter, accepting either a uuid.UUID or its
// string form so an HTTP query parameter can be passed straight through.
func auditUserFilter(filters map[string]interface{}) (uuid.UUID, bool) {
	raw, ok := filters["user_id"]
	if !ok {
		return uuid.Nil, false
	}

	switch v := raw.(type) {
	case uuid.UUID:
		return v, v != uuid.Nil
	case *uuid.UUID:
		if v == nil || *v == uuid.Nil {
			return uuid.Nil, false
		}
		return *v, true
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, false
		}
		return id, id != uuid.Nil
	default:
		return uuid.Nil, false
	}
}

func getUserIDFromContext(ctx context.Context) *uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return &userID
	}
	return nil
}

func getIPFromContext(ctx context.Context) string {
	if ip, ok := ctx.Value("ip_address").(string); ok {
		return ip
	}
	return ""
}

func getUserAgentFromContext(ctx context.Context) string {
	if ua, ok := ctx.Value("user_agent").(string); ok {
		return ua
	}
	return ""
}
