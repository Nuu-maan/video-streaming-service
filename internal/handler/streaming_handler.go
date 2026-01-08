package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/orchids/video-streaming/internal/config"
	"github.com/orchids/video-streaming/internal/domain"
	"github.com/orchids/video-streaming/internal/repository"
	"github.com/orchids/video-streaming/pkg/logger"
	"github.com/orchids/video-streaming/pkg/response"
	"github.com/redis/go-redis/v9"
)

type StreamingHandler struct {
	videoRepo   repository.VideoRepository
	redisClient *redis.Client
	config      *config.Config
	log         *logger.Logger
}

func NewStreamingHandler(
	videoRepo repository.VideoRepository,
	redisClient *redis.Client,
	config *config.Config,
	log *logger.Logger,
) *StreamingHandler {
	return &StreamingHandler{
		videoRepo:   videoRepo,
		redisClient: redisClient,
		config:      config,
		log:         log,
	}
}

func (h *StreamingHandler) ServeMasterPlaylist(c *gin.Context) {
	ctx := c.Request.Context()
	
	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		h.log.Error(ctx, "failed to get video", map[string]interface{}{
			"video_id": videoID,
			"error":    err.Error(),
		})
		response.InternalError(c, "Failed to retrieve video")
		return
	}

	if !video.HLSReady || video.HLSMasterPath == nil {
		response.Error(c, http.StatusNotFound, "HLS_NOT_READY", "HLS streaming not available for this video")
		return
	}

	cacheKey := fmt.Sprintf("playlist:%s:master", videoID.String())
	cached, err := h.redisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		h.servePlaylistContent(c, cached)
		return
	}

	masterPath := filepath.Join("./web/uploads/processed", videoID.String(), "hls", "master.m3u8")
	content, err := os.ReadFile(masterPath)
	if err != nil {
		h.log.Error(ctx, "failed to read master playlist", map[string]interface{}{
			"video_id": videoID,
			"path":     masterPath,
			"error":    err.Error(),
		})
		response.Error(c, http.StatusNotFound, "PLAYLIST_NOT_FOUND", "Playlist file not found")
		return
	}

	contentStr := string(content)
	h.redisClient.Set(ctx, cacheKey, contentStr, 1*time.Hour)

	h.servePlaylistContent(c, contentStr)
}

func (h *StreamingHandler) ServeQualityPlaylist(c *gin.Context) {
	ctx := c.Request.Context()
	
	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	quality := c.Param("quality")
	if !isValidQuality(quality) {
		response.ValidationError(c, "Invalid quality parameter")
		return
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		response.InternalError(c, "Failed to retrieve video")
		return
	}

	if !video.HLSReady {
		response.Error(c, http.StatusNotFound, "HLS_NOT_READY", "HLS streaming not available")
		return
	}

	cacheKey := fmt.Sprintf("playlist:%s:%s", videoID.String(), quality)
	cached, err := h.redisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		h.servePlaylistContent(c, cached)
		return
	}

	playlistPath := filepath.Join("./web/uploads/processed", videoID.String(), "hls", quality, "playlist.m3u8")
	content, err := os.ReadFile(playlistPath)
	if err != nil {
		h.log.Error(ctx, "failed to read quality playlist", map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"path":     playlistPath,
			"error":    err.Error(),
		})
		response.Error(c, http.StatusNotFound, "PLAYLIST_NOT_FOUND", "Playlist file not found")
		return
	}

	contentStr := string(content)
	h.redisClient.Set(ctx, cacheKey, contentStr, 1*time.Hour)

	h.servePlaylistContent(c, contentStr)
}

func (h *StreamingHandler) ServeSegment(c *gin.Context) {
	ctx := c.Request.Context()
	
	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	quality := c.Param("quality")
	if !isValidQuality(quality) {
		response.ValidationError(c, "Invalid quality parameter")
		return
	}

	segment := c.Param("segment")
	if !isValidSegmentName(segment) {
		response.ValidationError(c, "Invalid segment name")
		return
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		response.InternalError(c, "Failed to retrieve video")
		return
	}

	if !video.HLSReady {
		response.Error(c, http.StatusNotFound, "HLS_NOT_READY", "HLS streaming not available")
		return
	}

	segmentPath := filepath.Join("./web/uploads/processed", videoID.String(), "hls", quality, segment)
	
	fileInfo, err := os.Stat(segmentPath)
	if err != nil {
		h.log.Error(ctx, "segment not found", map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"segment":  segment,
			"path":     segmentPath,
		})
		response.Error(c, http.StatusNotFound, "SEGMENT_NOT_FOUND", "Segment file not found")
		return
	}

	c.Header("Content-Type", "video/MP2T")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	
	c.File(segmentPath)
}

func (h *StreamingHandler) ServeMP4Fallback(c *gin.Context) {
	ctx := c.Request.Context()
	
	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	quality := c.Param("quality")
	if !isValidQuality(quality) {
		response.ValidationError(c, "Invalid quality parameter")
		return
	}

	video, err := h.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			response.NotFound(c, "Video not found")
			return
		}
		response.InternalError(c, "Failed to retrieve video")
		return
	}

	if video.Status != domain.VideoStatusReady {
		response.Error(c, http.StatusNotFound, "VIDEO_NOT_READY", "Video not ready for streaming")
		return
	}

	qualityFound := false
	for _, q := range video.AvailableQualities {
		if q == quality {
			qualityFound = true
			break
		}
	}
	if !qualityFound {
		response.ValidationError(c, "Quality not available for this video")
		return
	}

	mp4Path := filepath.Join("./web/uploads/processed", videoID.String(), quality+".mp4")
	
	fileInfo, err := os.Stat(mp4Path)
	if err != nil {
		h.log.Error(ctx, "MP4 file not found", map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"path":     mp4Path,
		})
		response.Error(c, http.StatusNotFound, "FILE_NOT_FOUND", "Video file not found")
		return
	}

	c.Header("Content-Type", "video/mp4")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	
	c.File(mp4Path)
}

func (h *StreamingHandler) ClearPlaylistCache(c *gin.Context) {
	ctx := c.Request.Context()
	
	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.ValidationError(c, "Invalid video ID")
		return
	}

	pattern := fmt.Sprintf("playlist:%s:*", videoID.String())
	
	iter := h.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	deletedCount := 0
	for iter.Next(ctx) {
		if err := h.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			h.log.Error(ctx, "failed to delete cache key", map[string]interface{}{
				"key":   iter.Val(),
				"error": err.Error(),
			})
		} else {
			deletedCount++
		}
	}
	if err := iter.Err(); err != nil {
		h.log.Error(ctx, "cache scan error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Playlist cache cleared",
		"deleted": deletedCount,
	})
}

func (h *StreamingHandler) servePlaylistContent(c *gin.Context, content string) {
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
	c.String(http.StatusOK, content)
}

func isValidQuality(quality string) bool {
	validQualities := map[string]bool{
		"360p":  true,
		"480p":  true,
		"720p":  true,
		"1080p": true,
	}
	return validQualities[quality]
}

func isValidSegmentName(segment string) bool {
	matched, _ := regexp.MatchString(`^segment_\d{3}\.ts$`, segment)
	return matched
}
