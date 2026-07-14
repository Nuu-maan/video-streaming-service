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

// ContentReport is returned by the reporting and moderation endpoints, so its
// fields carry explicit snake_case tags: without them encoding/json emits the Go
// field names and this one type answers in PascalCase while the rest of the API
// answers in snake_case.
type ContentReport struct {
	ID          uuid.UUID    `json:"id"`
	VideoID     *uuid.UUID   `json:"video_id,omitempty"`
	UserID      *uuid.UUID   `json:"user_id,omitempty"`
	CommentID   *uuid.UUID   `json:"comment_id,omitempty"`
	ReporterID  uuid.UUID    `json:"reporter_id"`
	ReportType  ReportType   `json:"report_type"`
	Reason      string       `json:"reason"`
	Description string       `json:"description,omitempty"`
	Status      ReportStatus `json:"status"`
	ReviewedBy  *uuid.UUID   `json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time   `json:"reviewed_at,omitempty"`
	Action      *string      `json:"action,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
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
