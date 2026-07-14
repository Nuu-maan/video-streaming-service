package domain

import (
	"encoding/json"
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
	TranscodingProgress int             `json:"transcoding_progress"`
	AvailableQualities  []string        `json:"available_qualities"`
	HLSReady            bool            `json:"hls_ready"`
	StreamingProtocol   string          `json:"streaming_protocol,omitempty"`

	// ThumbnailPath and HLSMasterPath are storage keys, not URLs, and are
	// withheld from the API for the same reason as FilePath: they describe where
	// the bytes live on the server, which is nobody's business and is not
	// fetchable anyway. Clients get thumbnail_url and hls_url instead — see
	// MarshalJSON — which are real, access-controlled endpoints.
	ThumbnailPath *string `json:"-"`
	HLSMasterPath *string `json:"-"`

	// Discovery metadata. Search filters on these, so they are part of the
	// contract even though the upload endpoint leaves them empty by default.
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Language string   `json:"language,omitempty"`

	// Engagement counters. Postgres triggers maintain LikeCount and
	// CommentCount; ViewCount is incremented by the view tracker, which has no
	// trigger behind it. They are denormalised onto videos so a listing does not
	// need one aggregate query per row.
	ViewCount    int64 `json:"view_count"`
	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// videoJSON strips the MarshalJSON method so the marshaller below can embed the
// struct without recursing into itself.
type videoJSON Video

// MarshalJSON emits the playable URLs alongside the video.
//
// They are derived from the ID rather than stored, because a stored URL is a
// URL that goes stale: hls_master_path used to be persisted at transcode time
// and pointed at a directory the worker had never written to, so every API
// response advertised a link that 404'd. A client should be told where to fetch
// the video from, and that answer is a function of the route table, not of a
// column written months ago.
func (v Video) MarshalJSON() ([]byte, error) {
	out := struct {
		videoJSON
		ThumbnailURL string `json:"thumbnail_url,omitempty"`
		HLSURL       string `json:"hls_url,omitempty"`
	}{videoJSON: videoJSON(v)}

	if v.ThumbnailPath != nil && *v.ThumbnailPath != "" {
		out.ThumbnailURL = VideoThumbnailURL(v.ID)
	}
	if v.HLSReady {
		out.HLSURL = VideoHLSURL(v.ID)
	}

	return json.Marshal(out)
}

// VideoThumbnailURL and VideoHLSURL are the canonical client-facing URLs for a
// video's assets. They live here, next to the type they describe, so every
// projection of a video (the full record, a search hit, a playlist entry) agrees
// on one answer.
func VideoThumbnailURL(id uuid.UUID) string {
	return "/api/v1/videos/" + id.String() + "/thumbnail"
}

func VideoHLSURL(id uuid.UUID) string {
	return "/api/v1/videos/" + id.String() + "/hls/master.m3u8"
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
