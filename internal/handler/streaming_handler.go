package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Nuu-maan/video-streaming-service/internal/cache"
	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// playlistCacheTTL is how long a rendered HLS playlist stays cached. Playlists
// are immutable once a video is ready, so this is bounded only by the desire to
// eventually reclaim the space.
const playlistCacheTTL = time.Hour

type StreamingHandler struct {
	videoRepo repository.VideoRepository
	cache     *cache.CacheService
	config    *config.Config
	log       *logger.Logger
}

func NewStreamingHandler(
	videoRepo repository.VideoRepository,
	cacheService *cache.CacheService,
	config *config.Config,
	log *logger.Logger,
) *StreamingHandler {
	return &StreamingHandler{
		videoRepo: videoRepo,
		cache:     cacheService,
		config:    config,
		log:       log,
	}
}

// transcodedDir is where the worker writes a video's MP4 renditions and HLS
// output. It must agree with TranscodingService, which builds its output path
// from StorageConfig.TranscodedPath.
//
// These two disagreed: the worker wrote to TranscodedPath ("web/uploads/
// transcoded/<id>") while the streaming handler read from a hardcoded
// "./web/uploads/processed/<id>", a directory nothing ever created. Every
// playlist and segment request therefore 404'd, so HLS playback could never
// have worked. Both sides now derive the path from the same config value.
func (h *StreamingHandler) transcodedDir(videoID uuid.UUID) string {
	return filepath.Join(h.config.Storage.TranscodedPath, videoID.String())
}

// servePlaylist returns a cached playlist, falling back to reading it from disk
// and populating the cache. Both playlist endpoints had their own copy of this.
//
// A cache failure is not fatal: the file on disk is the source of truth, so a
// Redis outage degrades latency rather than availability.
func (h *StreamingHandler) servePlaylist(c *gin.Context, cacheKey, path string, fields map[string]interface{}) {
	ctx := c.Request.Context()

	cached, err := h.cache.Get(ctx, cacheKey)
	if err != nil {
		h.log.Warn(ctx, "playlist cache unavailable; reading from disk", fields)
	} else if len(cached) > 0 {
		h.servePlaylistContent(c, string(cached))
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		h.log.Error(ctx, "failed to read playlist", err, fields)
		response.Error(c, http.StatusNotFound, "PLAYLIST_NOT_FOUND", "Playlist file not found")
		return
	}

	if err := h.cache.Set(ctx, cacheKey, content, cache.CacheOptions{
		TTL:        playlistCacheTTL,
		LocalCache: true,
	}); err != nil {
		h.log.Warn(ctx, "could not cache playlist", fields)
	}

	h.servePlaylistContent(c, string(content))
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
		h.log.Error(ctx, "failed to get video", err, map[string]interface{}{
			"video_id": videoID,
		})
		response.InternalError(c, "Failed to retrieve video")
		return
	}

	if !video.HLSReady || video.HLSMasterPath == nil {
		response.Error(c, http.StatusNotFound, "HLS_NOT_READY", "HLS streaming not available for this video")
		return
	}

	masterPath := filepath.Join(h.transcodedDir(videoID), "hls", "master.m3u8")

	h.servePlaylist(c,
		fmt.Sprintf("playlist:%s:master", videoID),
		masterPath,
		map[string]interface{}{"video_id": videoID, "path": masterPath},
	)
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

	playlistPath := filepath.Join(h.transcodedDir(videoID), "hls", quality, "playlist.m3u8")

	h.servePlaylist(c,
		fmt.Sprintf("playlist:%s:%s", videoID, quality),
		playlistPath,
		map[string]interface{}{"video_id": videoID, "quality": quality, "path": playlistPath},
	)
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

	segmentPath := filepath.Join(h.transcodedDir(videoID), "hls", quality, segment)

	fileInfo, err := os.Stat(segmentPath)
	if err != nil {
		h.log.Error(ctx, "segment not found", err, map[string]interface{}{
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

	mp4Path := filepath.Join(h.transcodedDir(videoID), quality+".mp4")

	fileInfo, err := os.Stat(mp4Path)
	if err != nil {
		h.log.Error(ctx, "MP4 file not found", err, map[string]interface{}{
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

	pattern := fmt.Sprintf("playlist:%s:*", videoID)

	if err := h.cache.DeletePattern(ctx, pattern); err != nil {
		h.log.Error(ctx, "failed to clear playlist cache", err, map[string]interface{}{
			"video_id": videoID,
		})
		response.InternalError(c, "Failed to clear playlist cache")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message":  "Playlist cache cleared",
		"video_id": videoID,
	})
}

func (h *StreamingHandler) servePlaylistContent(c *gin.Context, content string) {
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "public, max-age=3600")
	// The CORS headers are deliberately not set here. This used to send
	// Access-Control-Allow-Origin: * on every playlist, silently overriding the
	// configured origin allowlist for exactly the routes a player calls. The
	// CORS middleware owns that policy for the whole API.
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
