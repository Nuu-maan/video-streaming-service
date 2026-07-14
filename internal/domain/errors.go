package domain

import "errors"

// Sentinel errors returned by the domain layer. Callers match them with
// errors.Is; the HTTP layer maps them to status codes. Never compare error
// strings.
var (
	// Generic.
	ErrInvalidInput  = errors.New("invalid input")
	ErrDatabaseError = errors.New("database error")

	// Video.
	ErrVideoNotFound    = errors.New("video not found")
	ErrInvalidTitle     = errors.New("invalid video title")
	ErrTitleTooLong     = errors.New("video title is too long")
	ErrInvalidFileSize  = errors.New("invalid file size")
	ErrFileSizeTooLarge = errors.New("file size exceeds maximum allowed")
	ErrInvalidFilename  = errors.New("invalid filename")
	ErrInvalidMimeType  = errors.New("invalid mime type")
	ErrInvalidFormat    = errors.New("invalid video format")
	ErrUploadFailed     = errors.New("video upload failed")
	ErrProcessingFailed = errors.New("video processing failed")
	ErrInvalidProgress  = errors.New("invalid progress value")
	ErrInvalidStatus    = errors.New("invalid video status")
	ErrInvalidID        = errors.New("invalid video ID")

	// User and authentication.
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidUsername    = errors.New("username must be 3-30 characters and contain only letters, numbers, and underscores")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidRole        = errors.New("invalid user role")
	ErrBioTooLong         = errors.New("bio exceeds 500 characters")
	ErrFullNameTooLong    = errors.New("full name exceeds 100 characters")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrWeakPassword       = errors.New("password does not meet security requirements")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrForbidden          = errors.New("forbidden access")
	ErrUserBanned         = errors.New("user is banned")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session has expired")

	// Moderation.
	ErrInvalidReportType   = errors.New("invalid report type")
	ErrMissingReportTarget = errors.New("report must have at least one target")
	ErrMissingReportReason = errors.New("report reason is required")
	ErrReportNotFound      = errors.New("report not found")

	// Social.
	ErrLikeNotFound           = errors.New("like not found")
	ErrCommentNotFound        = errors.New("comment not found")
	ErrSubscriptionNotFound   = errors.New("subscription not found")
	ErrSelfSubscription       = errors.New("cannot subscribe to yourself")
	ErrPlaylistNotFound       = errors.New("playlist not found")
	ErrPlaylistVideoNotFound  = errors.New("video is not in the playlist")
	ErrVideoAlreadyInPlaylist = errors.New("video is already in the playlist")
	ErrWatchLaterNotFound     = errors.New("video is not in watch later")
	ErrNotificationNotFound   = errors.New("notification not found")

	// Account email verification.
	ErrEmailAlreadyVerified = errors.New("email is already verified")

	// Token revocation.
	ErrTokenRevoked          = errors.New("token has been revoked")
	ErrRevocationUnavailable = errors.New("revocation state unavailable")

	// Watch history.
	ErrWatchHistoryNotFound = errors.New("watch history entry not found")

	// Storage.
	ErrStorageKeyInvalid     = errors.New("invalid storage key")
	ErrStorageObjectNotFound = errors.New("storage object not found")
)
