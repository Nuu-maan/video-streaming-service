package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/config"
	"github.com/orchids/video-streaming/internal/domain"
	"github.com/orchids/video-streaming/internal/repository"
	"github.com/orchids/video-streaming/pkg/logger"
	"github.com/orchids/video-streaming/pkg/validator"
)

type UploadService struct {
	videoRepo     repository.VideoRepository
	ffmpegService *FFmpegService
	config        *config.StorageConfig
	log           *logger.Logger
}

func NewUploadService(
	videoRepo repository.VideoRepository,
	ffmpegService *FFmpegService,
	config *config.StorageConfig,
	log *logger.Logger,
) *UploadService {
	return &UploadService{
		videoRepo:     videoRepo,
		ffmpegService: ffmpegService,
		config:        config,
		log:           log,
	}
}

func (s *UploadService) UploadVideo(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	title, description string,
) (*domain.Video, error) {
	s.log.Info(ctx, "starting video upload", map[string]interface{}{
		"filename": header.Filename,
		"size":     header.Size,
		"title":    title,
	})

	title = validator.SanitizeString(title)
	description = validator.SanitizeString(description)

	if err := validator.ValidateTitle(title); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := validator.ValidateDescription(description); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := validator.ValidateVideoFile(file, header, s.config.MaxUploadSize); err != nil {
		return nil, err
	}

	uniqueID := uuid.New()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	filename := fmt.Sprintf("%s%s", uniqueID.String(), ext)
	
	uploadDir := filepath.Join(s.config.UploadPath, "raw")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	filePath := filepath.Join(uploadDir, filename)
	
	destFile, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	written, err := io.Copy(destFile, file)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	if written != header.Size {
		os.Remove(filePath)
		return nil, fmt.Errorf("file size mismatch: expected %d, got %d", header.Size, written)
	}

	s.log.Info(ctx, "file saved to disk", map[string]interface{}{
		"path": filePath,
		"size": written,
	})

	metadata, err := s.ffmpegService.ExtractMetadata(ctx, filePath)
	if err != nil {
		s.log.Error(ctx, "failed to extract metadata, saving video anyway", map[string]interface{}{
			"error": err.Error(),
			"file":  filePath,
		})
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "video/" + strings.TrimPrefix(ext, ".")
	}

	video := &domain.Video{
		ID:          uniqueID,
		Title:       title,
		Description: &description,
		Filename:    filename,
		FilePath:    filePath,
		FileSize:    header.Size,
		MimeType:    mimeType,
		Status:      domain.StatusUploading,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if metadata != nil {
		video.Duration = &metadata.Duration
		resolution := fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
		video.OriginalResolution = &resolution
	}

	if err := s.videoRepo.Create(ctx, video); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save video metadata: %w", err)
	}

	s.log.Info(ctx, "video upload completed", map[string]interface{}{
		"video_id": video.ID,
		"title":    video.Title,
		"duration": video.Duration,
	})

	return video, nil
}
