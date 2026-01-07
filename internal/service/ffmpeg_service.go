package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/orchids/video-streaming/pkg/logger"
)

type VideoMetadata struct {
	Duration   float64
	Width      int
	Height     int
	FrameRate  float64
	Bitrate    int64
	VideoCodec string
	AudioCodec string
	Format     string
}

type FFmpegService struct {
	log            *logger.Logger
	ffprobePath    string
	ffprobePathMux sync.Once
}

func NewFFmpegService(log *logger.Logger) *FFmpegService {
	return &FFmpegService{
		log: log,
	}
}

func (s *FFmpegService) ExtractMetadata(ctx context.Context, filePath string) (*VideoMetadata, error) {
	s.ensureFFprobePath()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ffprobe timeout after 30 seconds")
		}
		return nil, fmt.Errorf("ffprobe execution failed: %w", err)
	}

	var probeData struct {
		Format struct {
			Duration string `json:"duration"`
			Bitrate  string `json:"bit_rate"`
			Format   string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			RFrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	metadata := &VideoMetadata{
		Format: probeData.Format.Format,
	}

	if dur, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
		metadata.Duration = dur
	}

	if br, err := strconv.ParseInt(probeData.Format.Bitrate, 10, 64); err == nil {
		metadata.Bitrate = br
	}

	for _, stream := range probeData.Streams {
		if stream.CodecType == "video" && metadata.VideoCodec == "" {
			metadata.VideoCodec = stream.CodecName
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			
			if stream.RFrameRate != "" && strings.Contains(stream.RFrameRate, "/") {
				parts := strings.Split(stream.RFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den > 0 {
						metadata.FrameRate = num / den
					}
				}
			}
		}
		if stream.CodecType == "audio" && metadata.AudioCodec == "" {
			metadata.AudioCodec = stream.CodecName
		}
	}

	if metadata.VideoCodec == "" {
		return nil, fmt.Errorf("no video stream found in file")
	}

	s.log.Info(ctx, "extracted video metadata", map[string]interface{}{
		"file":       filePath,
		"duration":   metadata.Duration,
		"resolution": fmt.Sprintf("%dx%d", metadata.Width, metadata.Height),
		"codec":      metadata.VideoCodec,
	})

	return metadata, nil
}

func (s *FFmpegService) ensureFFprobePath() {
	s.ffprobePathMux.Do(func() {
		path, err := exec.LookPath("ffprobe")
		if err != nil {
			s.ffprobePath = "ffprobe"
		} else {
			s.ffprobePath = path
		}
	})
}
