package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/config"
	"github.com/orchids/video-streaming/internal/domain"
	"github.com/orchids/video-streaming/internal/repository"
	"github.com/orchids/video-streaming/pkg/logger"
)

type QualitySpec struct {
	Name      string
	Width     int
	Height    int
	Bitrate   string
	MaxRate   string
	BufSize   string
	FPS       int
}

var qualitySpecs = map[string]QualitySpec{
	"360p": {
		Name:    "360p",
		Width:   640,
		Height:  360,
		Bitrate: "800k",
		MaxRate: "900k",
		BufSize: "1800k",
		FPS:     30,
	},
	"480p": {
		Name:    "480p",
		Width:   854,
		Height:  480,
		Bitrate: "1400k",
		MaxRate: "1500k",
		BufSize: "3000k",
		FPS:     30,
	},
	"720p": {
		Name:    "720p",
		Width:   1280,
		Height:  720,
		Bitrate: "2800k",
		MaxRate: "3000k",
		BufSize: "6000k",
		FPS:     30,
	},
	"1080p": {
		Name:    "1080p",
		Width:   1920,
		Height:  1080,
		Bitrate: "5000k",
		MaxRate: "5500k",
		BufSize: "11000k",
		FPS:     60,
	},
}

type TranscodingService struct {
	videoRepo     repository.VideoRepository
	ffmpegService *FFmpegService
	storage       *config.StorageConfig
	log           *logger.Logger
	ffmpegPath    string
	ffmpegPathMux sync.Once
}

func NewTranscodingService(
	videoRepo repository.VideoRepository,
	ffmpegService *FFmpegService,
	storage *config.StorageConfig,
	log *logger.Logger,
) *TranscodingService {
	return &TranscodingService{
		videoRepo:     videoRepo,
		ffmpegService: ffmpegService,
		storage:       storage,
		log:           log,
	}
}

func (s *TranscodingService) ProcessVideo(ctx context.Context, videoID string) error {
	s.log.Info(ctx, "starting video processing", map[string]interface{}{
		"video_id": videoID,
	})

	id, err := uuid.Parse(videoID)
	if err != nil {
		return fmt.Errorf("invalid video ID: %w", err)
	}

	video, err := s.videoRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get video: %w", err)
	}

	if video.Status != domain.VideoStatusUploading && video.Status != domain.VideoStatusFailed {
		s.log.Warn(ctx, "video is not in uploadable status", map[string]interface{}{
			"video_id": videoID,
			"status":   video.Status,
		})
		return fmt.Errorf("video status is %s, expected uploading or failed", video.Status)
	}

	if err := s.videoRepo.UpdateStatus(ctx, id, domain.VideoStatusProcessing); err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	metadata, err := s.ffmpegService.ExtractMetadata(ctx, video.FilePath)
	if err != nil {
		s.log.Error(ctx, "failed to extract metadata", map[string]interface{}{
			"video_id": videoID,
			"error":    err.Error(),
		})
		s.videoRepo.MarkAsFailed(ctx, id)
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	duration := int(metadata.Duration)
	resolution := fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
	
	if err := s.videoRepo.UpdateDuration(ctx, id, duration); err != nil {
		s.log.Error(ctx, "failed to update duration", map[string]interface{}{
			"video_id": videoID,
			"error":    err.Error(),
		})
	}
	
	if err := s.videoRepo.UpdateResolution(ctx, id, resolution); err != nil {
		s.log.Error(ctx, "failed to update resolution", map[string]interface{}{
			"video_id": videoID,
			"error":    err.Error(),
		})
	}

	outputDir := filepath.Join(s.storage.TranscodedPath, videoID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		s.videoRepo.MarkAsFailed(ctx, id)
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	qualities := []string{"360p", "480p", "720p", "1080p"}
	transcoded := []string{}
	totalSteps := len(qualities) + 1

	for i, quality := range qualities {
		spec := qualitySpecs[quality]
		
		if metadata.Height < spec.Height {
			s.log.Info(ctx, "skipping quality (would upscale)", map[string]interface{}{
				"video_id":          videoID,
				"quality":           quality,
				"original_height":   metadata.Height,
				"target_height":     spec.Height,
			})
			continue
		}

		progress := int(float64(i) / float64(totalSteps) * 100)
		if err := s.videoRepo.UpdateProgress(ctx, id, progress); err != nil {
			s.log.Error(ctx, "failed to update progress", map[string]interface{}{
				"video_id": videoID,
				"progress": progress,
				"error":    err.Error(),
			})
		}

		outputPath := filepath.Join(outputDir, quality+".mp4")
		if err := s.transcodeVideo(ctx, video.FilePath, outputPath, spec); err != nil {
			s.log.Error(ctx, "failed to transcode quality", map[string]interface{}{
				"video_id": videoID,
				"quality":  quality,
				"error":    err.Error(),
			})
			continue
		}

		transcoded = append(transcoded, quality)
		s.log.Info(ctx, "transcoded quality successfully", map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
		})
	}

	if len(transcoded) == 0 {
		s.videoRepo.MarkAsFailed(ctx, id)
		return fmt.Errorf("failed to transcode any quality")
	}

	thumbnailProgress := int(float64(len(qualities)) / float64(totalSteps) * 100)
	if err := s.videoRepo.UpdateProgress(ctx, id, thumbnailProgress); err != nil {
		s.log.Error(ctx, "failed to update progress", map[string]interface{}{
			"video_id": videoID,
			"progress": thumbnailProgress,
		})
	}

	thumbnailPath, err := s.generateThumbnail(ctx, video.FilePath, videoID, metadata.Duration)
	if err != nil {
		s.log.Error(ctx, "failed to generate thumbnail", map[string]interface{}{
			"video_id": videoID,
			"error":    err.Error(),
		})
		thumbnailPath = ""
	}

	if err := s.videoRepo.MarkAsReady(ctx, id, transcoded, thumbnailPath); err != nil {
		return fmt.Errorf("failed to mark video as ready: %w", err)
	}

	if err := s.videoRepo.UpdateProgress(ctx, id, 100); err != nil {
		s.log.Error(ctx, "failed to update final progress", map[string]interface{}{
			"video_id": videoID,
			"error":    err.Error(),
		})
	}

	s.log.Info(ctx, "video processing completed", map[string]interface{}{
		"video_id":  videoID,
		"qualities": transcoded,
	})

	return nil
}

func (s *TranscodingService) transcodeVideo(ctx context.Context, inputPath, outputPath string, spec QualitySpec) error {
	s.ensureFFmpegPath()

	scaleFilter := fmt.Sprintf("scale=%d:%d", spec.Width, spec.Height)
	
	args := []string{
		"-i", inputPath,
		"-vf", scaleFilter,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-b:v", spec.Bitrate,
		"-maxrate", spec.MaxRate,
		"-bufsize", spec.BufSize,
		"-r", strconv.Itoa(spec.FPS),
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (s *TranscodingService) generateThumbnail(ctx context.Context, inputPath, videoID string, duration float64) (string, error) {
	s.ensureFFmpegPath()

	thumbnailDir := s.storage.ThumbnailPath
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	thumbnailPath := filepath.Join(thumbnailDir, videoID+".jpg")
	
	seekTime := duration * 0.1
	if seekTime > 10 {
		seekTime = 10
	}

	args := []string{
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.2f", seekTime),
		"-vframes", "1",
		"-vf", "scale=320:180",
		"-y",
		thumbnailPath,
	}

	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg thumbnail generation failed: %w, output: %s", err, string(output))
	}

	relPath := filepath.Join("thumbnails", videoID+".jpg")
	return relPath, nil
}

func (s *TranscodingService) ensureFFmpegPath() {
	s.ffmpegPathMux.Do(func() {
		path, err := exec.LookPath("ffmpeg")
		if err != nil {
			s.ffmpegPath = "ffmpeg"
		} else {
			s.ffmpegPath = path
		}
	})
}
