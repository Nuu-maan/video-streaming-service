package service

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type VideoQuality struct {
	Name       string
	Width      int
	Height     int
	Bitrate    string
	MaxRate    string
	BufSize    string
	AudioRate  string
	CRF        int
	Preset     string
}

var QualityPresets = map[string]VideoQuality{
	"360p": {
		Name:      "360p",
		Width:     640,
		Height:    360,
		Bitrate:   "800k",
		MaxRate:   "856k",
		BufSize:   "1200k",
		AudioRate: "96k",
		CRF:       28,
		Preset:    "slow",
	},
	"480p": {
		Name:      "480p",
		Width:     854,
		Height:    480,
		Bitrate:   "1400k",
		MaxRate:   "1498k",
		BufSize:   "2100k",
		AudioRate: "128k",
		CRF:       26,
		Preset:    "slow",
	},
	"720p": {
		Name:      "720p",
		Width:     1280,
		Height:    720,
		Bitrate:   "2800k",
		MaxRate:   "2996k",
		BufSize:   "4200k",
		AudioRate: "128k",
		CRF:       23,
		Preset:    "medium",
	},
	"1080p": {
		Name:      "1080p",
		Width:     1920,
		Height:    1080,
		Bitrate:   "5000k",
		MaxRate:   "5350k",
		BufSize:   "7500k",
		AudioRate: "192k",
		CRF:       21,
		Preset:    "medium",
	},
}

type VideoOptimizer struct {
	ffmpegPath  string
	ffprobePath string
	hwAccel     string
}

func NewVideoOptimizer(ffmpegPath, ffprobePath string) *VideoOptimizer {
	optimizer := &VideoOptimizer{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}

	optimizer.hwAccel = optimizer.detectHardwareAcceleration()

	return optimizer
}

func (o *VideoOptimizer) detectHardwareAcceleration() string {
	cmd := exec.Command(o.ffmpegPath, "-hwaccels")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	hwaccels := string(output)

	if strings.Contains(hwaccels, "cuda") {
		return "cuda"
	}
	if strings.Contains(hwaccels, "qsv") {
		return "qsv"
	}
	if strings.Contains(hwaccels, "videotoolbox") {
		return "videotoolbox"
	}

	return ""
}

func (o *VideoOptimizer) GetTranscodeArgs(inputFile, outputFile string, quality VideoQuality) []string {
	args := []string{
		"-i", inputFile,
		"-y",
	}

	if o.hwAccel == "cuda" {
		args = append(args,
			"-hwaccel", "cuda",
			"-hwaccel_output_format", "cuda",
		)
	}

	args = append(args,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			quality.Width, quality.Height, quality.Width, quality.Height),
	)

	if o.hwAccel == "cuda" {
		args = append(args,
			"-c:v", "h264_nvenc",
			"-preset", "p4",
			"-rc", "vbr",
			"-cq", fmt.Sprintf("%d", quality.CRF),
			"-b:v", quality.Bitrate,
			"-maxrate", quality.MaxRate,
			"-bufsize", quality.BufSize,
		)
	} else {
		args = append(args,
			"-c:v", "libx264",
			"-preset", quality.Preset,
			"-crf", fmt.Sprintf("%d", quality.CRF),
			"-profile:v", "main",
			"-level", "4.0",
			"-b:v", quality.Bitrate,
			"-maxrate", quality.MaxRate,
			"-bufsize", quality.BufSize,
		)
	}

	args = append(args,
		"-c:a", "aac",
		"-b:a", quality.AudioRate,
		"-ac", "2",
		"-ar", "44100",
	)

	args = append(args,
		"-movflags", "+faststart",
		"-pix_fmt", "yuv420p",
	)

	args = append(args, outputFile)

	return args
}

func (o *VideoOptimizer) GetHLSArgs(inputFile, outputDir string, quality VideoQuality, segmentDuration int) []string {
	playlistName := fmt.Sprintf("%s.m3u8", quality.Name)
	segmentPattern := fmt.Sprintf("%s_%%03d.ts", quality.Name)

	args := []string{
		"-i", inputFile,
		"-y",
	}

	if o.hwAccel == "cuda" {
		args = append(args,
			"-hwaccel", "cuda",
			"-hwaccel_output_format", "cuda",
		)
	}

	args = append(args,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			quality.Width, quality.Height, quality.Width, quality.Height),
	)

	if o.hwAccel == "cuda" {
		args = append(args,
			"-c:v", "h264_nvenc",
			"-preset", "p4",
			"-rc", "vbr",
			"-cq", fmt.Sprintf("%d", quality.CRF),
			"-b:v", quality.Bitrate,
			"-maxrate", quality.MaxRate,
			"-bufsize", quality.BufSize,
		)
	} else {
		args = append(args,
			"-c:v", "libx264",
			"-preset", quality.Preset,
			"-crf", fmt.Sprintf("%d", quality.CRF),
			"-profile:v", "main",
			"-level", "4.0",
			"-b:v", quality.Bitrate,
			"-maxrate", quality.MaxRate,
			"-bufsize", quality.BufSize,
		)
	}

	args = append(args,
		"-c:a", "aac",
		"-b:a", quality.AudioRate,
		"-ac", "2",
		"-ar", "44100",
	)

	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", segmentDuration),
		"-hls_list_size", "0",
		"-hls_segment_type", "mpegts",
		"-hls_segment_filename", filepath.Join(outputDir, segmentPattern),
		"-hls_playlist_type", "vod",
		"-pix_fmt", "yuv420p",
		filepath.Join(outputDir, playlistName),
	)

	return args
}

func (o *VideoOptimizer) GetTwoPassArgs(inputFile, outputFile string, quality VideoQuality, pass int) []string {
	args := []string{
		"-i", inputFile,
		"-y",
	}

	args = append(args,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			quality.Width, quality.Height, quality.Width, quality.Height),
		"-c:v", "libx264",
		"-preset", quality.Preset,
		"-b:v", quality.Bitrate,
		"-maxrate", quality.MaxRate,
		"-bufsize", quality.BufSize,
		"-profile:v", "main",
		"-level", "4.0",
		"-pass", fmt.Sprintf("%d", pass),
	)

	if pass == 1 {
		args = append(args,
			"-an",
			"-f", "null",
		)
		if isWindows() {
			args = append(args, "NUL")
		} else {
			args = append(args, "/dev/null")
		}
	} else {
		args = append(args,
			"-c:a", "aac",
			"-b:a", quality.AudioRate,
			"-ac", "2",
			"-ar", "44100",
			"-movflags", "+faststart",
			"-pix_fmt", "yuv420p",
			outputFile,
		)
	}

	return args
}

func (o *VideoOptimizer) GetThumbnailArgs(inputFile, outputFile string, timestamp string, width, height int) []string {
	return []string{
		"-i", inputFile,
		"-ss", timestamp,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			width, height, width, height),
		"-q:v", "2",
		"-y",
		outputFile,
	}
}

func (o *VideoOptimizer) GetWebPThumbnailArgs(inputFile, outputFile string, timestamp string, width, height int) []string {
	return []string{
		"-i", inputFile,
		"-ss", timestamp,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			width, height, width, height),
		"-c:v", "libwebp",
		"-quality", "80",
		"-y",
		outputFile,
	}
}

func (o *VideoOptimizer) GetAnimatedThumbnailArgs(inputFile, outputFile string, startTime string, duration int, width, height int) []string {
	return []string{
		"-i", inputFile,
		"-ss", startTime,
		"-t", fmt.Sprintf("%d", duration),
		"-vf", fmt.Sprintf("fps=10,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			width, height, width, height),
		"-c:v", "libwebp",
		"-loop", "0",
		"-quality", "60",
		"-y",
		outputFile,
	}
}

func (o *VideoOptimizer) GetProbeArgs(inputFile string) []string {
	return []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputFile,
	}
}

func (o *VideoOptimizer) GetAudioExtractArgs(inputFile, outputFile string) []string {
	return []string{
		"-i", inputFile,
		"-vn",
		"-c:a", "aac",
		"-b:a", "192k",
		"-y",
		outputFile,
	}
}

func (o *VideoOptimizer) GetSubtitleExtractArgs(inputFile, outputFile string, streamIndex int) []string {
	return []string{
		"-i", inputFile,
		"-map", fmt.Sprintf("0:%d", streamIndex),
		"-c:s", "webvtt",
		"-y",
		outputFile,
	}
}

func (o *VideoOptimizer) GetConcatArgs(listFile, outputFile string) []string {
	return []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y",
		outputFile,
	}
}

func (o *VideoOptimizer) GetTrimArgs(inputFile, outputFile string, startTime, endTime string) []string {
	return []string{
		"-i", inputFile,
		"-ss", startTime,
		"-to", endTime,
		"-c", "copy",
		"-y",
		outputFile,
	}
}

func (o *VideoOptimizer) HardwareAccelEnabled() bool {
	return o.hwAccel != ""
}

func (o *VideoOptimizer) HardwareAccelType() string {
	return o.hwAccel
}

func isWindows() bool {
	return filepath.Separator == '\\'
}
