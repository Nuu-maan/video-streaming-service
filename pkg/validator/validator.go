package validator

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrFileTooLarge   = fmt.Errorf("file size exceeds maximum allowed size")
	ErrInvalidFormat  = fmt.Errorf("invalid file format")
	ErrInvalidTitle   = fmt.Errorf("invalid title")
	ErrCorruptVideo   = fmt.Errorf("corrupt or invalid video file")
	ErrInvalidUUID    = fmt.Errorf("invalid UUID format")
	ErrInvalidPagination = fmt.Errorf("invalid pagination parameters")
)

var allowedExtensions = map[string]bool{
	".mp4":  true,
	".mov":  true,
	".avi":  true,
	".mkv":  true,
	".webm": true,
}

var videoMagicBytes = map[string][]byte{
	"mp4":  {0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, // ftyp
	"webm": {0x1A, 0x45, 0xDF, 0xA3},                         // EBML
	"avi":  {0x52, 0x49, 0x46, 0x46},                         // RIFF
}

func ValidateVideoFile(file multipart.File, header *multipart.FileHeader, maxSize int64) error {
	if header.Size > maxSize {
		return fmt.Errorf("%w: file is %d bytes, maximum is %d bytes", ErrFileTooLarge, header.Size, maxSize)
	}

	if header.Size < 1024 {
		return fmt.Errorf("%w: file is too small to be a valid video", ErrInvalidFormat)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		return fmt.Errorf("%w: only mp4, mov, avi, mkv, webm are allowed", ErrInvalidFormat)
	}

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file header: %w", err)
	}
	
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	if !isVideoFile(buf[:n]) {
		return fmt.Errorf("%w: file content does not match video format", ErrInvalidFormat)
	}

	return nil
}

func isVideoFile(buf []byte) bool {
	if len(buf) < 8 {
		return false
	}

	for _, magic := range videoMagicBytes {
		if len(buf) >= len(magic) {
			match := true
			for i, b := range magic {
				if buf[i] != b {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}

	return false
}

func ValidateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("%w: title cannot be empty", ErrInvalidTitle)
	}
	if len(title) > 255 {
		return fmt.Errorf("%w: title cannot exceed 255 characters", ErrInvalidTitle)
	}
	return nil
}

func ValidateDescription(description string) error {
	if len(description) > 5000 {
		return fmt.Errorf("description cannot exceed 5000 characters")
	}
	return nil
}

func ValidateUUID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", ErrInvalidUUID, id)
	}
	return parsed, nil
}

func ValidatePageParams(page, limit int) error {
	if page < 1 {
		return fmt.Errorf("%w: page must be >= 1", ErrInvalidPagination)
	}
	if limit < 1 || limit > 100 {
		return fmt.Errorf("%w: limit must be between 1 and 100", ErrInvalidPagination)
	}
	return nil
}

func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}
