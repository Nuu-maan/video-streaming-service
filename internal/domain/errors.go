package domain

import "errors"

var (
	ErrVideoNotFound      = errors.New("video not found")
	ErrInvalidTitle       = errors.New("invalid video title")
	ErrTitleTooLong       = errors.New("video title is too long")
	ErrInvalidFileSize    = errors.New("invalid file size")
	ErrFileSizeTooLarge   = errors.New("file size exceeds maximum allowed")
	ErrInvalidFilename    = errors.New("invalid filename")
	ErrInvalidMimeType    = errors.New("invalid mime type")
	ErrInvalidFormat      = errors.New("invalid video format")
	ErrUploadFailed       = errors.New("video upload failed")
	ErrProcessingFailed   = errors.New("video processing failed")
	ErrInvalidProgress    = errors.New("invalid progress value")
	ErrInvalidStatus      = errors.New("invalid video status")
	ErrDatabaseError      = errors.New("database error")
	ErrInvalidID          = errors.New("invalid video ID")
)
