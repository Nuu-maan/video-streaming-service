package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/domain"
)

type AuditLogRepository interface {
	CreateLog(ctx context.Context, log *domain.AuditLog) error
	GetLogs(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.AuditLog, int64, error)
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
		if err := s.repo.CreateLog(context.Background(), log); err != nil {
		}
	}()

	return nil
}

func (s *AuditService) GetLogs(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.AuditLog, int64, error) {
	return s.repo.GetLogs(ctx, filters, limit, offset)
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
