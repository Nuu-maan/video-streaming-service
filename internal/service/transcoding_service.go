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
	totalSteps := len(qualities) + 2

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

	hlsProgress := int(float64(len(qualities)) / float64(totalSteps) * 100)
	if err := s.videoRepo.UpdateProgress(ctx, id, hlsProgress); err != nil {
		s.log.Error(ctx, "failed to update progress", map[string]interface{}{
			"video_id": videoID,
			"progress": hlsProgress,
		})
	}

	hlsQualities := []string{}
	for _, quality := range transcoded {
		mp4Path := filepath.Join(outputDir, quality+".mp4")
		if err := s.ConvertToHLS(ctx, videoID, quality, mp4Path); err != nil {
			s.log.Error(ctx, "failed to convert to HLS", map[string]interface{}{
				"video_id": videoID,
				"quality":  quality,
				"error":    err.Error(),
			})
			continue
		}
		hlsQualities = append(hlsQualities, quality)
	}

	if len(hlsQualities) > 0 {
		if err := s.GenerateMasterPlaylist(ctx, videoID, hlsQualities); err != nil {
			s.log.Error(ctx, "failed to generate master playlist", map[string]interface{}{
				"video_id": videoID,
				"error":    err.Error(),
			})
		} else {
			hlsMasterPath := fmt.Sprintf("/uploads/processed/%s/hls/master.m3u8", videoID)
			if err := s.videoRepo.UpdateHLSInfo(ctx, id, hlsMasterPath, true); err != nil {
				s.log.Error(ctx, "failed to update HLS info", map[string]interface{}{
					"video_id": videoID,
					"error":    err.Error(),
				})
			}
		}
	}

	thumbnailProgress := int(float64(len(qualities)+1) / float64(totalSteps) * 100)
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

func (s *TranscodingService) ConvertToHLS(ctx context.Context, videoID, quality, mp4Path string) error {
	s.ensureFFmpegPath()

	hlsDir := filepath.Join(s.storage.TranscodedPath, videoID, "hls", quality)
	if err := os.MkdirAll(hlsDir, 0755); err != nil {
		return fmt.Errorf("failed to create HLS directory: %w", err)
	}

	playlistPath := filepath.Join(hlsDir, "playlist.m3u8")
	segmentPattern := filepath.Join(hlsDir, "segment_%03d.ts")

	args := []string{
		"-i", mp4Path,
		"-c:v", "copy",
		"-c:a", "copy",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
		"-hls_list_size", "0",
		"-y",
		playlistPath,
	}

	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
			return fmt.Errorf("HLS conversion cancelled or timed out")
		}
		
		s.log.Error(ctx, "HLS conversion failed, retrying once", map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"error":    err.Error(),
			"output":   string(output),
		})
		
		time.Sleep(2 * time.Second)
		cmd = exec.CommandContext(ctx, s.ffmpegPath, args...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("HLS conversion failed after retry: %w, output: %s", err, string(output))
		}
	}

	segmentFiles, err := filepath.Glob(filepath.Join(hlsDir, "segment_*.ts"))
	if err != nil {
		return fmt.Errorf("failed to verify segments: %w", err)
	}
	if len(segmentFiles) == 0 {
		return fmt.Errorf("no segments generated for quality %s", quality)
	}

	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		return fmt.Errorf("playlist file not created: %s", playlistPath)
	}

	s.log.Info(ctx, "HLS conversion successful", map[string]interface{}{
		"video_id":       videoID,
		"quality":        quality,
		"segment_count":  len(segmentFiles),
		"playlist_path":  playlistPath,
	})

	return nil
}

func (s *TranscodingService) GenerateMasterPlaylist(ctx context.Context, videoID string, qualities []string) error {
	hlsBaseDir := filepath.Join(s.storage.TranscodedPath, videoID, "hls")
	masterPath := filepath.Join(hlsBaseDir, "master.m3u8")

	file, err := os.Create(masterPath)
	if err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "#EXTM3U")
	fmt.Fprintln(file, "#EXT-X-VERSION:3")

	bandwidthMap := map[string]int{
		"360p":  800000,
		"480p":  1400000,
		"720p":  2800000,
		"1080p": 5000000,
	}

	resolutionMap := map[string]string{
		"360p":  "640x360",
		"480p":  "854x480",
		"720p":  "1280x720",
		"1080p": "1920x1080",
	}

	for _, quality := range qualities {
		playlistPath := filepath.Join(hlsBaseDir, quality, "playlist.m3u8")
		if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
			s.log.Warn(ctx, "quality playlist not found, skipping", map[string]interface{}{
				"video_id": videoID,
				"quality":  quality,
			})
			continue
		}

		bandwidth := bandwidthMap[quality]
		resolution := resolutionMap[quality]

		fmt.Fprintf(file, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s\n", bandwidth, resolution)
		fmt.Fprintf(file, "%s/playlist.m3u8\n", quality)
	}

	s.log.Info(ctx, "master playlist generated", map[string]interface{}{
		"video_id":   videoID,
		"qualities":  qualities,
		"master_path": masterPath,
	})

	return nil
}
