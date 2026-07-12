package domain

import (
	"time"

	"github.com/google/uuid"
)

type VideoStatus string

const (
	VideoStatusUploading  VideoStatus = "uploading"
	VideoStatusProcessing VideoStatus = "processing"
	VideoStatusReady      VideoStatus = "ready"
	VideoStatusFailed     VideoStatus = "failed"
)

// VideoVisibility mirrors the video_visibility enum introduced in migration 4.
type VideoVisibility string

const (
	VisibilityPublic   VideoVisibility = "public"
	VisibilityPrivate  VideoVisibility = "private"
	VisibilityUnlisted VideoVisibility = "unlisted"
)

// IsValid reports whether v is a known visibility value.
func (v VideoVisibility) IsValid() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate, VisibilityUnlisted:
		return true
	default:
		return false
	}
}

// Video is an uploaded video and its transcoding state.
//
// Fields carry explicit snake_case json tags so the serialized shape is part of
// the API contract rather than an accident of Go field naming. FilePath is
// withheld: it is a server-side filesystem path and leaking it tells a caller
// exactly where the bytes live on disk.
type Video struct {
	ID uuid.UUID `json:"id"`
	// UserID is the owner. It is nil for videos uploaded before authentication
	// existed, so ownership checks must treat nil as "no owner" rather than
	// dereferencing it.
	UserID              *uuid.UUID      `json:"user_id,omitempty"`
	Title               string          `json:"title"`
	Description         string          `json:"description"`
	Filename            string          `json:"filename"`
	FilePath            string          `json:"-"`
	FileSize            int64           `json:"file_size"`
	Duration            int             `json:"duration"`
	Status              VideoStatus     `json:"status"`
	Visibility          VideoVisibility `json:"visibility"`
	MimeType            string          `json:"mime_type"`
	OriginalResolution  string          `json:"original_resolution,omitempty"`
	ThumbnailPath       *string         `json:"thumbnail_path,omitempty"`
	TranscodingProgress int             `json:"transcoding_progress"`
	AvailableQualities  []string        `json:"available_qualities"`
	HLSMasterPath       *string         `json:"hls_master_path,omitempty"`
	HLSReady            bool            `json:"hls_ready"`
	StreamingProtocol   string          `json:"streaming_protocol,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	ProcessedAt         *time.Time      `json:"processed_at,omitempty"`
}

// IsOwnedBy reports whether userID owns this video. An unowned (legacy) video
// is owned by nobody.
func (v *Video) IsOwnedBy(userID uuid.UUID) bool {
	return v.UserID != nil && *v.UserID == userID
}

// IsPubliclyVisible reports whether the video may be listed to anonymous users.
func (v *Video) IsPubliclyVisible() bool {
	return v.Visibility == VisibilityPublic && v.Status == VideoStatusReady
}

func NewVideo(title, description, filename, filePath, mimeType string, fileSize int64) (*Video, error) {
	video := &Video{
		ID:                  uuid.New(),
		Title:               title,
		Description:         description,
		Filename:            filename,
		FilePath:            filePath,
		FileSize:            fileSize,
		MimeType:            mimeType,
		Status:              VideoStatusUploading,
		Visibility:          VisibilityPublic,
		TranscodingProgress: 0,
		AvailableQualities:  []string{},
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := video.Validate(); err != nil {
		return nil, err
	}

	return video, nil
}

func (v *Video) IsProcessing() bool {
	return v.Status == VideoStatusProcessing
}

func (v *Video) CanBeStreamed() bool {
	return v.Status == VideoStatusReady && len(v.AvailableQualities) > 0
}

func (v *Video) MarkAsProcessing() {
	v.Status = VideoStatusProcessing
	v.TranscodingProgress = 0
	v.UpdatedAt = time.Now()
}

func (v *Video) MarkAsReady(qualities []string, thumbnailPath string) {
	v.Status = VideoStatusReady
	v.AvailableQualities = qualities
	v.ThumbnailPath = &thumbnailPath
	v.TranscodingProgress = 100
	now := time.Now()
	v.ProcessedAt = &now
	v.UpdatedAt = now
}

func (v *Video) MarkAsFailed() {
	v.Status = VideoStatusFailed
	v.UpdatedAt = time.Now()
}

func (v *Video) UpdateProgress(percent int) error {
	if percent < 0 || percent > 100 {
		return ErrInvalidProgress
	}
	v.TranscodingProgress = percent
	v.UpdatedAt = time.Now()
	return nil
}

func (v *Video) Validate() error {
	if v.Title == "" {
		return ErrInvalidTitle
	}
	if len(v.Title) > 255 {
		return ErrTitleTooLong
	}
	if v.FileSize <= 0 {
		return ErrInvalidFileSize
	}
	if v.FileSize > 10*1024*1024*1024 {
		return ErrFileSizeTooLarge
	}
	if v.Filename == "" {
		return ErrInvalidFilename
	}
	if v.MimeType == "" {
		return ErrInvalidMimeType
	}
	return nil
}
