package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewVideo(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		filename string
		mimeType string
		fileSize int64
		wantErr  error
	}{
		{
			name:     "valid video",
			title:    "My Holiday",
			filename: "holiday.mp4",
			mimeType: "video/mp4",
			fileSize: 1024,
			wantErr:  nil,
		},
		{
			name:     "empty title",
			title:    "",
			filename: "holiday.mp4",
			mimeType: "video/mp4",
			fileSize: 1024,
			wantErr:  ErrInvalidTitle,
		},
		{
			name:     "title too long",
			title:    strings.Repeat("a", 256),
			filename: "holiday.mp4",
			mimeType: "video/mp4",
			fileSize: 1024,
			wantErr:  ErrTitleTooLong,
		},
		{
			name:     "title at max length is accepted",
			title:    strings.Repeat("a", 255),
			filename: "holiday.mp4",
			mimeType: "video/mp4",
			fileSize: 1024,
			wantErr:  nil,
		},
		{
			name:     "zero file size",
			title:    "My Holiday",
			filename: "holiday.mp4",
			mimeType: "video/mp4",
			fileSize: 0,
			wantErr:  ErrInvalidFileSize,
		},
		{
			name:     "negative file size",
			title:    "My Holiday",
			filename: "holiday.mp4",
			mimeType: "video/mp4",
			fileSize: -1,
			wantErr:  ErrInvalidFileSize,
		},
		{
			name:     "file size too large",
			title:    "My Holiday",
			filename: "holiday.mp4",
			mimeType: "video/mp4",
			fileSize: 10*1024*1024*1024 + 1,
			wantErr:  ErrFileSizeTooLarge,
		},
		{
			name:     "empty filename",
			title:    "My Holiday",
			filename: "",
			mimeType: "video/mp4",
			fileSize: 1024,
			wantErr:  ErrInvalidFilename,
		},
		{
			name:     "empty mime type",
			title:    "My Holiday",
			filename: "holiday.mp4",
			mimeType: "",
			fileSize: 1024,
			wantErr:  ErrInvalidMimeType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video, err := NewVideo(tt.title, "a description", tt.filename, "/videos/holiday.mp4", tt.mimeType, tt.fileSize)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewVideo() error = %v, want %v", err, tt.wantErr)
				}
				if video != nil {
					t.Errorf("NewVideo() returned a video alongside an error: %+v", video)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewVideo() unexpected error: %v", err)
			}
			if video == nil {
				t.Fatal("NewVideo() returned nil video with nil error")
			}
			if video.Title != tt.title {
				t.Errorf("Title = %q, want %q", video.Title, tt.title)
			}
			if video.FileSize != tt.fileSize {
				t.Errorf("FileSize = %d, want %d", video.FileSize, tt.fileSize)
			}
		})
	}
}

func TestNewVideoDefaults(t *testing.T) {
	video, err := NewVideo("Title", "Description", "file.mp4", "/videos/file.mp4", "video/mp4", 2048)
	if err != nil {
		t.Fatalf("NewVideo() unexpected error: %v", err)
	}

	if video.Status != VideoStatusUploading {
		t.Errorf("Status = %q, want %q", video.Status, VideoStatusUploading)
	}
	if video.Visibility != VisibilityPublic {
		t.Errorf("Visibility = %q, want %q", video.Visibility, VisibilityPublic)
	}
	if video.ID == uuid.Nil {
		t.Error("ID was not generated")
	}
	if video.UserID != nil {
		t.Errorf("UserID = %v, want nil for a video created without an owner", video.UserID)
	}
	if video.TranscodingProgress != 0 {
		t.Errorf("TranscodingProgress = %d, want 0", video.TranscodingProgress)
	}
	if video.AvailableQualities == nil || len(video.AvailableQualities) != 0 {
		t.Errorf("AvailableQualities = %v, want empty non-nil slice", video.AvailableQualities)
	}
	if video.CreatedAt.IsZero() || video.UpdatedAt.IsZero() {
		t.Error("CreatedAt/UpdatedAt were not set")
	}
}

func TestVideoIsOwnedBy(t *testing.T) {
	owner := uuid.New()
	other := uuid.New()

	tests := []struct {
		name    string
		videoBy *uuid.UUID
		asker   uuid.UUID
		want    bool
	}{
		{
			name:    "owner",
			videoBy: &owner,
			asker:   owner,
			want:    true,
		},
		{
			name:    "different user",
			videoBy: &owner,
			asker:   other,
			want:    false,
		},
		{
			// Legacy videos predate authentication and carry no owner. This must
			// report false rather than panicking on a nil dereference.
			name:    "unowned legacy video",
			videoBy: nil,
			asker:   owner,
			want:    false,
		},
		{
			name:    "unowned legacy video, zero asker",
			videoBy: nil,
			asker:   uuid.Nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video := &Video{UserID: tt.videoBy}

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("IsOwnedBy() panicked: %v", r)
				}
			}()

			if got := video.IsOwnedBy(tt.asker); got != tt.want {
				t.Errorf("IsOwnedBy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVideoIsPubliclyVisible(t *testing.T) {
	tests := []struct {
		name       string
		visibility VideoVisibility
		status     VideoStatus
		want       bool
	}{
		{"public and ready", VisibilityPublic, VideoStatusReady, true},
		{"public but uploading", VisibilityPublic, VideoStatusUploading, false},
		{"public but processing", VisibilityPublic, VideoStatusProcessing, false},
		{"public but failed", VisibilityPublic, VideoStatusFailed, false},
		{"private and ready", VisibilityPrivate, VideoStatusReady, false},
		{"unlisted and ready", VisibilityUnlisted, VideoStatusReady, false},
		{"private and processing", VisibilityPrivate, VideoStatusProcessing, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video := &Video{Visibility: tt.visibility, Status: tt.status}
			if got := video.IsPubliclyVisible(); got != tt.want {
				t.Errorf("IsPubliclyVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVideoVisibilityIsValid(t *testing.T) {
	tests := []struct {
		name       string
		visibility VideoVisibility
		want       bool
	}{
		{"public", VisibilityPublic, true},
		{"private", VisibilityPrivate, true},
		{"unlisted", VisibilityUnlisted, true},
		{"empty", VideoVisibility(""), false},
		{"unknown", VideoVisibility("secret"), false},
		{"wrong case", VideoVisibility("PUBLIC"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.visibility.IsValid(); got != tt.want {
				t.Errorf("VideoVisibility(%q).IsValid() = %v, want %v", tt.visibility, got, tt.want)
			}
		})
	}
}
