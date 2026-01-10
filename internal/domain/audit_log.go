package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID
	UserID     *uuid.UUID
	Action     string
	TargetType string
	TargetID   *uuid.UUID
	IPAddress  string
	UserAgent  string
	Details    map[string]interface{}
	CreatedAt  time.Time
}

func NewAuditLog(userID *uuid.UUID, action, targetType string, targetID *uuid.UUID, ipAddress, userAgent string, details map[string]interface{}) *AuditLog {
	return &AuditLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Details:    details,
		CreatedAt:  time.Now(),
	}
}

const (
	ActionUserLogin         = "user.login"
	ActionUserLogout        = "user.logout"
	ActionUserRegister      = "user.register"
	ActionUserUpdate        = "user.update"
	ActionUserDelete        = "user.delete"
	ActionUserBan           = "user.ban"
	ActionUserUnban         = "user.unban"
	ActionUserRoleChange    = "user.role_change"
	ActionVideoUpload       = "video.upload"
	ActionVideoUpdate       = "video.update"
	ActionVideoDelete       = "video.delete"
	ActionVideoView         = "video.view"
	ActionReportCreate      = "report.create"
	ActionReportReview      = "report.review"
	ActionReportResolve     = "report.resolve"
	ActionReportDismiss     = "report.dismiss"
	ActionSystemAlert       = "system.alert"
	ActionSystemBackup      = "system.backup"
)
