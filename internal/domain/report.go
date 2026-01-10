package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReportType string

const (
	ReportTypeSpam           ReportType = "spam"
	ReportTypeHarassment     ReportType = "harassment"
	ReportTypeHateSpeech     ReportType = "hate_speech"
	ReportTypeViolence       ReportType = "violence"
	ReportTypeCopyright      ReportType = "copyright"
	ReportTypeNudity         ReportType = "nudity"
	ReportTypeMisinformation ReportType = "misinformation"
	ReportTypeOther          ReportType = "other"
)

func (rt ReportType) IsValid() bool {
	switch rt {
	case ReportTypeSpam, ReportTypeHarassment, ReportTypeHateSpeech,
		ReportTypeViolence, ReportTypeCopyright, ReportTypeNudity,
		ReportTypeMisinformation, ReportTypeOther:
		return true
	}
	return false
}

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewing ReportStatus = "reviewing"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

func (rs ReportStatus) IsValid() bool {
	switch rs {
	case ReportStatusPending, ReportStatusReviewing, ReportStatusResolved, ReportStatusDismissed:
		return true
	}
	return false
}

type ContentReport struct {
	ID          uuid.UUID
	VideoID     *uuid.UUID
	UserID      *uuid.UUID
	CommentID   *uuid.UUID
	ReporterID  uuid.UUID
	ReportType  ReportType
	Reason      string
	Description string
	Status      ReportStatus
	ReviewedBy  *uuid.UUID
	ReviewedAt  *time.Time
	Action      *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewContentReport(reporterID uuid.UUID, reportType ReportType, reason, description string) (*ContentReport, error) {
	report := &ContentReport{
		ID:          uuid.New(),
		ReporterID:  reporterID,
		ReportType:  reportType,
		Reason:      reason,
		Description: description,
		Status:      ReportStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := report.Validate(); err != nil {
		return nil, err
	}

	return report, nil
}

func (r *ContentReport) Validate() error {
	if !r.ReportType.IsValid() {
		return ErrInvalidReportType
	}
	if r.VideoID == nil && r.UserID == nil && r.CommentID == nil {
		return ErrMissingReportTarget
	}
	if r.Reason == "" {
		return ErrMissingReportReason
	}
	return nil
}

func (r *ContentReport) MarkAsReviewing(moderatorID uuid.UUID) {
	r.Status = ReportStatusReviewing
	r.ReviewedBy = &moderatorID
	r.UpdatedAt = time.Now()
}

func (r *ContentReport) Resolve(moderatorID uuid.UUID, action string) {
	r.Status = ReportStatusResolved
	r.ReviewedBy = &moderatorID
	now := time.Now()
	r.ReviewedAt = &now
	r.Action = &action
	r.UpdatedAt = now
}

func (r *ContentReport) Dismiss(moderatorID uuid.UUID) {
	r.Status = ReportStatusDismissed
	r.ReviewedBy = &moderatorID
	now := time.Now()
	r.ReviewedAt = &now
	r.UpdatedAt = now
}

type ModerationResult struct {
	ContentID       uuid.UUID
	ContentType     string
	Flagged         bool
	Confidence      float64
	Violations      []string
	SuggestedAction string
	CreatedAt       time.Time
}
