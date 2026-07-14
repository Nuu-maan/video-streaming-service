package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/google/uuid"
)

// ErrDuplicateReport is a sentinel so the HTTP layer can answer a re-report with
// a conflict rather than a 500: reporting the same video twice is a client
// mistake, not a server fault.
var ErrDuplicateReport = errors.New("user has already reported this content")

// ModerationRepository is the slice of the report store that moderation needs.
// It is stated in the method names the postgres implementation actually has, so
// *postgres.ReportRepository satisfies it directly with no adapter.
type ModerationRepository interface {
	Create(ctx context.Context, report *domain.ContentReport) error
	GetByID(ctx context.Context, reportID uuid.UUID) (*domain.ContentReport, error)
	GetPending(ctx context.Context, limit, offset int) ([]*domain.ContentReport, error)
	Update(ctx context.Context, report *domain.ContentReport) error
	CountByStatus(ctx context.Context, status domain.ReportStatus) (int64, error)
	CountByReporterAndVideo(ctx context.Context, reporterID, videoID uuid.UUID) (int64, error)
}

// VideoRepository is the slice of the video store that moderation needs.
// Satisfied by *postgres.PostgresVideoRepository.
type VideoRepository interface {
	Delete(ctx context.Context, videoID uuid.UUID) error
	GetByID(ctx context.Context, videoID uuid.UUID) (*domain.Video, error)
}

// UserRepository is the slice of the user store that moderation needs.
// Satisfied by *postgres.UserRepository.
type UserRepository interface {
	BanUser(ctx context.Context, userID uuid.UUID, reason string, banExpiry *time.Time) error
	UnbanUser(ctx context.Context, userID uuid.UUID) error
	GetByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

// VideoFileRemover cleans a deleted video's files out of storage. Satisfied by
// *UploadService. A nil remover is tolerated so tests that only exercise report
// bookkeeping need no storage.
type VideoFileRemover interface {
	RemoveVideoFiles(ctx context.Context, video *domain.Video)
}

type ModerationService struct {
	reportRepo ModerationRepository
	videoRepo  VideoRepository
	userRepo   UserRepository
	files      VideoFileRemover
	auditSvc   *AuditService
}

func NewModerationService(
	reportRepo ModerationRepository,
	videoRepo VideoRepository,
	userRepo UserRepository,
	files VideoFileRemover,
	auditSvc *AuditService,
) *ModerationService {
	return &ModerationService{
		reportRepo: reportRepo,
		videoRepo:  videoRepo,
		userRepo:   userRepo,
		files:      files,
		auditSvc:   auditSvc,
	}
}

func (s *ModerationService) CreateReport(ctx context.Context, report *domain.ContentReport) error {
	if err := report.Validate(); err != nil {
		return err
	}

	// Duplicate detection only applies to video reports; a report whose target is
	// a user or a comment has no video ID to key on.
	if report.VideoID != nil {
		existingCount, err := s.reportRepo.CountByReporterAndVideo(ctx, report.ReporterID, *report.VideoID)
		if err != nil {
			return fmt.Errorf("failed to check existing reports: %w", err)
		}

		if existingCount > 0 {
			return ErrDuplicateReport
		}
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
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

// GetPendingReports returns one page of pending reports plus the total number of
// pending reports, so a caller can render pagination.
func (s *ModerationService) GetPendingReports(ctx context.Context, limit, offset int) ([]*domain.ContentReport, int64, error) {
	reports, err := s.reportRepo.GetPending(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing pending reports: %w", err)
	}

	total, err := s.reportRepo.CountByStatus(ctx, domain.ReportStatusPending)
	if err != nil {
		return nil, 0, fmt.Errorf("counting pending reports: %w", err)
	}

	return reports, total, nil
}

func (s *ModerationService) ReviewReport(ctx context.Context, reportID, moderatorID uuid.UUID, action, notes string) error {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return fmt.Errorf("failed to get report: %w", err)
	}

	switch action {
	case "delete_video":
		if report.VideoID != nil {
			// Loaded before the delete: the row carries the file path the
			// storage cleanup needs, and it is gone afterwards.
			video, err := s.videoRepo.GetByID(ctx, *report.VideoID)
			if err != nil {
				return fmt.Errorf("looking up reported video: %w", err)
			}
			if err := s.videoRepo.Delete(ctx, *report.VideoID); err != nil {
				return fmt.Errorf("failed to delete video: %w", err)
			}
			if s.files != nil {
				s.files.RemoveVideoFiles(ctx, video)
			}
		}
		report.Resolve(moderatorID, action)

	case "ban_user":
		var targetUserID uuid.UUID
		switch {
		case report.UserID != nil:
			targetUserID = *report.UserID
		case report.VideoID != nil:
			video, err := s.videoRepo.GetByID(ctx, *report.VideoID)
			if err != nil {
				return fmt.Errorf("looking up reported video: %w", err)
			}
			// Videos uploaded before authentication existed have no owner, so
			// there is nobody to ban.
			if video.UserID == nil {
				return fmt.Errorf("%w: reported video has no owner to ban", domain.ErrInvalidInput)
			}
			targetUserID = *video.UserID
		default:
			return domain.ErrMissingReportTarget
		}

		if err := s.userRepo.BanUser(ctx, targetUserID, notes, nil); err != nil {
			return fmt.Errorf("banning user: %w", err)
		}
		report.Resolve(moderatorID, action)

	case "warn_user":
		report.Resolve(moderatorID, action)

	case "dismiss":
		report.Dismiss(moderatorID)

	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	if err := s.reportRepo.Update(ctx, report); err != nil {
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
