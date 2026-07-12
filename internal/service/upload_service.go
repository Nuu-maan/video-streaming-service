package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
)

// UploadService persists an uploaded video to storage and records it in the
// repository. Transcoding is not its concern: it leaves the video in
// VideoStatusUploading for the worker to pick up.
type UploadService struct {
	videoRepo     repository.VideoRepository
	ffmpegService *FFmpegService
	storage       *config.StorageConfig
	log           *logger.Logger
}

func NewUploadService(
	videoRepo repository.VideoRepository,
	ffmpegService *FFmpegService,
	storage *config.StorageConfig,
	log *logger.Logger,
) *UploadService {
	return &UploadService{
		videoRepo:     videoRepo,
		ffmpegService: ffmpegService,
		storage:       storage,
		log:           log,
	}
}

// UploadRequest describes a single video upload.
type UploadRequest struct {
	File        multipart.File
	Header      *multipart.FileHeader
	Title       string
	Description string
	OwnerID     uuid.UUID
	Visibility  domain.VideoVisibility
}

// UploadVideo validates the request, streams the file to disk, probes it for
// metadata, and records it. If anything fails after the file lands on disk, the
// file is removed: a half-written upload with no database row is a leak.
func (s *UploadService) UploadVideo(ctx context.Context, req UploadRequest) (video *domain.Video, err error) {
	title := validator.SanitizeString(req.Title)
	description := validator.SanitizeString(req.Description)

	if err := validator.ValidateTitle(title); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidTitle, err)
	}
	if err := validator.ValidateDescription(description); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}
	if err := validator.ValidateVideoFile(req.File, req.Header, s.storage.MaxFileSize); err != nil {
		return nil, err
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = domain.VisibilityPublic
	}
	if !visibility.IsValid() {
		return nil, fmt.Errorf("%w: unknown visibility %q", domain.ErrInvalidInput, visibility)
	}

	filePath, err := s.persistFile(req.File, req.Header)
	if err != nil {
		return nil, err
	}
	// Any failure past this point must not leave an orphaned file behind.
	defer func() {
		if err != nil {
			if removeErr := os.Remove(filePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				s.log.Error(ctx, "failed to clean up orphaned upload", removeErr, map[string]interface{}{
					"path": filePath,
				})
			}
		}
	}()

	video, err = domain.NewVideo(title, description, filepath.Base(filePath), filePath, mimeTypeOf(req.Header), req.Header.Size)
	if err != nil {
		return nil, err
	}
	video.Visibility = visibility
	if req.OwnerID != uuid.Nil {
		owner := req.OwnerID
		video.UserID = &owner
	}

	// A video that will not probe is still worth storing; the transcoding
	// worker probes again and fails the video properly if it is truly corrupt.
	if metadata, probeErr := s.ffmpegService.ExtractMetadata(ctx, filePath); probeErr != nil {
		s.log.Warn(ctx, "could not probe uploaded video; storing without metadata", map[string]interface{}{
			"path":  filePath,
			"error": probeErr.Error(),
		})
	} else {
		video.Duration = int(metadata.Duration)
		video.OriginalResolution = fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
	}

	if err = s.videoRepo.Create(ctx, video); err != nil {
		return nil, fmt.Errorf("recording video: %w", err)
	}

	s.log.Info(ctx, "video upload completed", map[string]interface{}{
		"video_id": video.ID,
		"size":     video.FileSize,
		"duration": video.Duration,
	})

	return video, nil
}

// persistFile streams the upload to disk under a generated name and returns its
// path. The stored name is derived from a fresh UUID, never from user input, so
// a crafted filename cannot escape the upload directory.
func (s *UploadService) persistFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	uploadDir := filepath.Join(s.storage.UploadPath, "raw")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", fmt.Errorf("creating upload directory: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	filePath := filepath.Join(uploadDir, uuid.New().String()+ext)

	dest, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("creating destination file: %w", err)
	}

	written, err := io.Copy(dest, file)
	if err != nil {
		dest.Close()
		os.Remove(filePath)
		return "", fmt.Errorf("writing upload to disk: %w", err)
	}

	// Close before returning: on Windows the file cannot be removed while open,
	// and a deferred Close would also swallow the flush error.
	if err := dest.Close(); err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("flushing upload to disk: %w", err)
	}

	if written != header.Size {
		os.Remove(filePath)
		return "", fmt.Errorf("%w: expected %d bytes, wrote %d", domain.ErrUploadFailed, header.Size, written)
	}

	return filePath, nil
}

// mimeTypeOf trusts the browser-supplied content type only as a fallback label;
// validator.ValidateVideoFile has already sniffed the magic bytes.
func mimeTypeOf(header *multipart.FileHeader) string {
	if mimeType := header.Header.Get("Content-Type"); mimeType != "" {
		return mimeType
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(header.Filename)), ".")
	return "video/" + ext
}
