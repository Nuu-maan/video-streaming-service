package validator

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrFileTooLarge       = fmt.Errorf("file size exceeds maximum allowed size")
	ErrInvalidFormat      = fmt.Errorf("invalid file format")
	ErrInvalidTitle       = fmt.Errorf("invalid title")
	ErrCorruptVideo       = fmt.Errorf("corrupt or invalid video file")
	ErrInvalidUUID        = fmt.Errorf("invalid UUID format")
	ErrInvalidPagination  = fmt.Errorf("invalid pagination parameters")
	ErrInvalidDescription = fmt.Errorf("invalid description")
)

var allowedExtensions = map[string]bool{
	".mp4":  true,
	".mov":  true,
	".avi":  true,
	".mkv":  true,
	".webm": true,
}

// Container signatures, in the byte order they appear at the head of the file.
var (
	// magicEBML starts every Matroska and WebM file (.mkv, .webm).
	magicEBML = []byte{0x1A, 0x45, 0xDF, 0xA3}
	// magicRIFF and magicAVI bracket the AVI form type: "RIFF????AVI ".
	magicRIFF = []byte{'R', 'I', 'F', 'F'}
	magicAVI  = []byte{'A', 'V', 'I', ' '}
	// magicFTYP is the ISO Base Media File Format brand box, shared by .mp4,
	// .mov, and .m4v. It sits at offset 4, after a 4-byte box size.
	magicFTYP = []byte{'f', 't', 'y', 'p'}
)

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

// isVideoFile reports whether buf begins with a recognised video container
// signature.
//
// The ISO-BMFF check matches "ftyp" at offset 4 and ignores the four preceding
// box-size bytes. The previous version required those bytes to be exactly
// 00 00 00 18, which is only one of many legal box sizes — so most real-world
// .mp4 files were rejected as corrupt. It also had no signature at all for .mov
// (an ISO-BMFF format), even though .mov is in the extension allowlist, making
// every .mov upload fail. Matroska (.mkv) shares the EBML signature with WebM.
func isVideoFile(buf []byte) bool {
	if len(buf) >= 8 && bytes.Equal(buf[4:8], magicFTYP) {
		return true // mp4, mov, m4v
	}
	if bytes.HasPrefix(buf, magicEBML) {
		return true // mkv, webm
	}
	if len(buf) >= 12 && bytes.HasPrefix(buf, magicRIFF) && bytes.Equal(buf[8:12], magicAVI) {
		return true // avi
	}
	return false
}

// Text arriving over HTTP is a byte string, not a Go string in any meaningful
// sense: nothing has checked that it is valid UTF-8. Postgres does check, and
// refuses the insert with SQLSTATE 22021 ("invalid byte sequence for encoding
// UTF8"), which surfaced to the client as a 500 — an internal error for what is
// plainly bad input. A form submitted in Windows-1252 is enough to trigger it: a
// single em dash arrives as the byte 0x97, which is not valid UTF-8.
//
// Length is counted in runes, not bytes. `len(s)` counts bytes, so a title in
// Japanese — three bytes per character — was refused at around 85 characters
// even though the column holds 255. Postgres counts characters; so do we.
func validateText(value string, maxRunes int, field string, sentinel error) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8 text", sentinel, field)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return fmt.Errorf("%w: %s cannot exceed %d characters", sentinel, field, maxRunes)
	}
	return nil
}

func ValidateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("%w: title cannot be empty", ErrInvalidTitle)
	}
	return validateText(title, 255, "title", ErrInvalidTitle)
}

func ValidateDescription(description string) error {
	return validateText(description, 5000, "description", ErrInvalidDescription)
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
