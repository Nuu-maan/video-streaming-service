package domain

import (
	"fmt"
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

type Video struct {
	ID                   uuid.UUID
	Title                string
	Description          string
	Filename             string
	FilePath             string
	FileSize             int64
	Duration             int
	Status               VideoStatus
	MimeType             string
	OriginalResolution   string
	ThumbnailPath        *string
	TranscodingProgress  int
	AvailableQualities   []string
	HLSMasterPath        *string
	HLSReady             bool
	StreamingProtocol    string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ProcessedAt          *time.Time
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
