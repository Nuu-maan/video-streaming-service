package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"time"

	"github.com/Nuu-maan/video-streaming-service/internal/cache"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/internal/storage"
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
	store     storage.Store
	log       *logger.Logger
}

func NewStreamingHandler(
	videoRepo repository.VideoRepository,
	cacheService *cache.CacheService,
	store storage.Store,
	log *logger.Logger,
) *StreamingHandler {
	return &StreamingHandler{
		videoRepo: videoRepo,
		cache:     cacheService,
		store:     store,
		log:       log,
	}
}

// transcodedKey addresses a file under a video's transcoded output in the
// store. The worker writes under this same layout (the local store maps the
// "transcoded" area onto StorageConfig.TranscodedPath), so writer and reader
// agree by construction. They used to disagree — the worker wrote to
// TranscodedPath while this handler read a hardcoded "processed" directory
// nothing ever created, so every playlist and segment request 404'd.
func transcodedKey(videoID uuid.UUID, parts ...string) string {
	return storage.Key(append([]string{"transcoded", videoID.String()}, parts...)...)
}

// servePlaylist returns a cached playlist, falling back to reading it from the
// store and populating the cache. Both playlist endpoints had their own copy of
// this.
//
// A cache failure is not fatal: the stored file is the source of truth, so a
// Redis outage degrades latency rather than availability.
func (h *StreamingHandler) servePlaylist(c *gin.Context, cacheKey, key string, fields map[string]interface{}) {
	ctx := c.Request.Context()

	cached, err := h.cache.Get(ctx, cacheKey)
	if err != nil {
		h.log.Warn(ctx, "playlist cache unavailable; reading from storage", fields)
	} else if len(cached) > 0 {
		h.servePlaylistContent(c, string(cached))
		return
	}

	content, err := h.readObject(ctx, key)
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

// readObject slurps a whole object from the store. Only suitable for
// playlists, which are a few hundred bytes; segments stream instead.
func (h *StreamingHandler) readObject(ctx context.Context, key string) ([]byte, error) {
	obj, err := h.store.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	content, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", key, err)
	}
	return content, nil
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

	if !canViewVideo(ctx, video) {
		response.NotFound(c, "Video not found")
		return
	}

	if !video.HLSReady || video.HLSMasterPath == nil {
		response.Error(c, http.StatusNotFound, "HLS_NOT_READY", "HLS streaming not available for this video")
		return
	}

	masterKey := transcodedKey(videoID, "hls", "master.m3u8")

	h.servePlaylist(c,
		fmt.Sprintf("playlist:%s:master", videoID),
		masterKey,
		map[string]interface{}{"video_id": videoID, "key": masterKey},
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

	if !canViewVideo(ctx, video) {
		response.NotFound(c, "Video not found")
		return
	}

	if !video.HLSReady {
		response.Error(c, http.StatusNotFound, "HLS_NOT_READY", "HLS streaming not available")
		return
	}

	playlistKey := transcodedKey(videoID, "hls", quality, "playlist.m3u8")

	h.servePlaylist(c,
		fmt.Sprintf("playlist:%s:%s", videoID, quality),
		playlistKey,
		map[string]interface{}{"video_id": videoID, "quality": quality, "key": playlistKey},
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

	if !canViewVideo(ctx, video) {
		response.NotFound(c, "Video not found")
		return
	}

	if !video.HLSReady {
		response.Error(c, http.StatusNotFound, "HLS_NOT_READY", "HLS streaming not available")
		return
	}

	segmentKey := transcodedKey(videoID, "hls", quality, segment)

	fileInfo, err := h.store.Stat(ctx, segmentKey)
	if err != nil {
		h.log.Error(ctx, "segment not found", err, map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"segment":  segment,
			"key":      segmentKey,
		})
		response.Error(c, http.StatusNotFound, "SEGMENT_NOT_FOUND", "Segment file not found")
		return
	}

	obj, err := h.store.Open(ctx, segmentKey)
	if err != nil {
		h.log.Error(ctx, "failed to open segment", err, map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"segment":  segment,
			"key":      segmentKey,
		})
		response.Error(c, http.StatusNotFound, "SEGMENT_NOT_FOUND", "Segment file not found")
		return
	}
	defer obj.Close()

	c.Header("Content-Type", "video/MP2T")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size))

	http.ServeContent(c.Writer, c.Request, segment, fileInfo.ModTime, obj)
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

	if !canViewVideo(ctx, video) {
		response.NotFound(c, "Video not found")
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

	mp4Key := transcodedKey(videoID, quality+".mp4")

	fileInfo, err := h.store.Stat(ctx, mp4Key)
	if err != nil {
		h.log.Error(ctx, "MP4 file not found", err, map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"key":      mp4Key,
		})
		response.Error(c, http.StatusNotFound, "FILE_NOT_FOUND", "Video file not found")
		return
	}

	obj, err := h.store.Open(ctx, mp4Key)
	if err != nil {
		h.log.Error(ctx, "failed to open MP4 file", err, map[string]interface{}{
			"video_id": videoID,
			"quality":  quality,
			"key":      mp4Key,
		})
		response.Error(c, http.StatusNotFound, "FILE_NOT_FOUND", "Video file not found")
		return
	}
	defer obj.Close()

	c.Header("Content-Type", "video/mp4")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size))

	// ServeContent needs only a ReadSeeker, so both backends can honour Range
	// requests; a plain io.Copy here would break seeking in every player.
	http.ServeContent(c.Writer, c.Request, quality+".mp4", fileInfo.ModTime, obj)
}

// ServeThumbnail serves a video's poster image.
//
// It exists because a thumbnail is not public data: it is a frame of the video,
// so a private video's thumbnail must be as private as the video. Serving it
// from a static file route would hand it to anyone who could guess the video ID,
// and would not work at all against object storage.
func (h *StreamingHandler) ServeThumbnail(c *gin.Context) {
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
		response.InternalError(c, "Failed to retrieve video")
		return
	}

	if !canViewVideo(ctx, video) {
		response.NotFound(c, "Video not found")
		return
	}

	if video.ThumbnailPath == nil || *video.ThumbnailPath == "" {
		response.NotFound(c, "Thumbnail not available")
		return
	}

	key := *video.ThumbnailPath

	fileInfo, err := h.store.Stat(ctx, key)
	if err != nil {
		response.NotFound(c, "Thumbnail not available")
		return
	}

	obj, err := h.store.Open(ctx, key)
	if err != nil {
		h.log.Error(ctx, "failed to open thumbnail", err, map[string]interface{}{
			"video_id": videoID,
			"key":      key,
		})
		response.NotFound(c, "Thumbnail not available")
		return
	}
	defer obj.Close()

	c.Header("Content-Type", "image/jpeg")
	// A thumbnail never changes once written: the worker generates it exactly
	// once, and a re-transcode produces a new video ID.
	c.Header("Cache-Control", "public, max-age=86400, immutable")
	http.ServeContent(c.Writer, c.Request, path.Base(key), fileInfo.ModTime, obj)
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
