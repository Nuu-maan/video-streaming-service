package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/domain"
)

type ModerationRepository interface {
	CreateReport(ctx context.Context, report *domain.ContentReport) error
	GetReportByID(ctx context.Context, reportID uuid.UUID) (*domain.ContentReport, error)
	GetPendingReports(ctx context.Context, limit, offset int) ([]*domain.ContentReport, int64, error)
	UpdateReport(ctx context.Context, report *domain.ContentReport) error
	GetUserReportCount(ctx context.Context, reporterID, targetID uuid.UUID) (int64, error)
}

type VideoRepository interface {
	DeleteVideo(ctx context.Context, videoID uuid.UUID) error
	GetVideoByID(ctx context.Context, videoID uuid.UUID) (*domain.Video, error)
}

type UserRepository interface {
	BanUser(ctx context.Context, userID uuid.UUID, reason string, banExpiry *time.Time) error
	UnbanUser(ctx context.Context, userID uuid.UUID) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

type ModerationService struct {
	reportRepo ModerationRepository
	videoRepo  VideoRepository
	userRepo   UserRepository
	auditSvc   *AuditService
}

func NewModerationService(
	reportRepo ModerationRepository,
	videoRepo VideoRepository,
	userRepo UserRepository,
	auditSvc *AuditService,
) *ModerationService {
	return &ModerationService{
		reportRepo: reportRepo,
		videoRepo:  videoRepo,
		userRepo:   userRepo,
		auditSvc:   auditSvc,
	}
}

func (s *ModerationService) CreateReport(ctx context.Context, report *domain.ContentReport) error {
	if err := report.Validate(); err != nil {
		return err
	}

	existingCount, err := s.reportRepo.GetUserReportCount(ctx, report.ReporterID, *report.VideoID)
	if err != nil {
		return fmt.Errorf("failed to check existing reports: %w", err)
	}

	if existingCount > 0 {
		return fmt.Errorf("user has already reported this content")
	}

	if err := s.reportRepo.CreateReport(ctx, report); err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	if err := s.auditSvc.Log(ctx, domain.ActionReportCreate, "report", &report.ID, map[string]interface{}{
		"report_type": report.ReportType,
		"target_type": getReportTargetType(report),
	}); err != nil {
		return fmt.Errorf("failed to log audit: %w", err)
	}

	return nil
}

func (s *ModerationService) GetPendingReports(ctx context.Context, limit, offset int) ([]*domain.ContentReport, int64, error) {
	return s.reportRepo.GetPendingReports(ctx, limit, offset)
}

func (s *ModerationService) ReviewReport(ctx context.Context, reportID, moderatorID uuid.UUID, action, notes string) error {
	report, err := s.reportRepo.GetReportByID(ctx, reportID)
	if err != nil {
		return fmt.Errorf("failed to get report: %w", err)
	}

	switch action {
	case "delete_video":
		if report.VideoID != nil {
			if err := s.videoRepo.DeleteVideo(ctx, *report.VideoID); err != nil {
				return fmt.Errorf("failed to delete video: %w", err)
			}
		}
		report.Resolve(moderatorID, action)

	case "ban_user":
		var targetUserID uuid.UUID
		if report.UserID != nil {
			targetUserID = *report.UserID
		} else if report.VideoID != nil {
			video, err := s.videoRepo.GetVideoByID(ctx, *report.VideoID)
			if err != nil {
				return fmt.Errorf("failed to get video: %w", err)
			}
			targetUserID = video.UserID
		}

		if err := s.userRepo.BanUser(ctx, targetUserID, notes, nil); err != nil {
			return fmt.Errorf("failed to ban user: %w", err)
		}
		report.Resolve(moderatorID, action)

	case "warn_user":
		report.Resolve(moderatorID, action)

	case "dismiss":
		report.Dismiss(moderatorID)

	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	if err := s.reportRepo.UpdateReport(ctx, report); err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	if err := s.auditSvc.Log(ctx, domain.ActionReportReview, "report", &reportID, map[string]interface{}{
		"action":       action,
		"notes":        notes,
		"moderator_id": moderatorID,
	}); err != nil {
		return fmt.Errorf("failed to log audit: %w", err)
	}

	return nil
}

func (s *ModerationService) BanUser(ctx context.Context, userID, moderatorID uuid.UUID, reason string, duration *time.Duration) error {
	var banExpiry *time.Time
	if duration != nil {
		expiry := time.Now().Add(*duration)
		banExpiry = &expiry
	}

	if err := s.userRepo.BanUser(ctx, userID, reason, banExpiry); err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}

	if err := s.auditSvc.Log(ctx, domain.ActionUserBan, "user", &userID, map[string]interface{}{
		"reason":       reason,
		"duration":     duration,
		"moderator_id": moderatorID,
	}); err != nil {
		return fmt.Errorf("failed to log audit: %w", err)
	}

	return nil
}

func (s *ModerationService) UnbanUser(ctx context.Context, userID, moderatorID uuid.UUID) error {
	if err := s.userRepo.UnbanUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to unban user: %w", err)
	}

	if err := s.auditSvc.Log(ctx, domain.ActionUserUnban, "user", &userID, map[string]interface{}{
		"moderator_id": moderatorID,
	}); err != nil {
		return fmt.Errorf("failed to log audit: %w", err)
	}

	return nil
}

func getReportTargetType(report *domain.ContentReport) string {
	if report.VideoID != nil {
		return "video"
	}
	if report.UserID != nil {
		return "user"
	}
	if report.CommentID != nil {
		return "comment"
	}
	return "unknown"
}
