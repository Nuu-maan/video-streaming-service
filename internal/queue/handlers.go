package queue

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/internal/service"
	"github.com/Nuu-maan/video-streaming-service/internal/storage"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

type VideoProcessingHandler struct {
	transcodingService *service.TranscodingService
	videoRepo          repository.VideoRepository
	store              storage.Store
	storageCfg         *config.StorageConfig
	logger             *logger.Logger
}

func NewVideoProcessingHandler(
	transcodingService *service.TranscodingService,
	videoRepo repository.VideoRepository,
	store storage.Store,
	storageCfg *config.StorageConfig,
	logger *logger.Logger,
) *VideoProcessingHandler {
	return &VideoProcessingHandler{
		transcodingService: transcodingService,
		videoRepo:          videoRepo,
		store:              store,
		storageCfg:         storageCfg,
		logger:             logger,
	}
}

// ProcessTask transcodes one video. ffmpeg reads and writes plain files, so
// transcoding always happens against the local paths in StorageConfig; when
// the store is remote, the raw file is staged from it first and the finished
// renditions are uploaded back afterwards. With a local store neither transfer
// happens — the working files already are the served files.
func (h *VideoProcessingHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	payload, err := ParseVideoProcessingPayload(task)
	if err != nil {
		h.logger.Error(ctx, "failed to parse video processing payload", err, map[string]interface{}{})
		return fmt.Errorf("parse payload: %w", err)
	}

	h.logger.Info(ctx, "processing video task", map[string]interface{}{
		"video_id":  payload.VideoID,
		"qualities": payload.Qualities,
		"priority":  payload.Priority,
		"task_id":   task.ResultWriter().TaskID(),
	})

	id, err := uuid.Parse(payload.VideoID)
	if err != nil {
		return fmt.Errorf("invalid video ID: %w", err)
	}

	remote := storage.IsRemote(h.store)

	var video *domain.Video
	if remote {
		video, err = h.videoRepo.GetByID(ctx, id)
		if err != nil {
			h.logger.Error(ctx, "failed to load video", err, map[string]interface{}{
				"video_id": payload.VideoID,
			})
			return fmt.Errorf("load video: %w", err)
		}
	}

	// A retried task can find the video already transcoded because a previous
	// attempt failed while uploading the outputs. ProcessVideo refuses a
	// "ready" video, so resume at the upload step instead of failing forever.
	if !remote || video.Status != domain.VideoStatusReady {
		if remote {
			if err := h.stageRawFile(ctx, video); err != nil {
				h.logger.Error(ctx, "failed to stage raw video", err, map[string]interface{}{
					"video_id": payload.VideoID,
				})
				return fmt.Errorf("stage raw video: %w", err)
			}
		}

		if err := h.transcodingService.ProcessVideo(ctx, payload.VideoID); err != nil {
			h.logger.Error(ctx, "video processing failed", err, map[string]interface{}{
				"video_id": payload.VideoID,
				"task_id":  task.ResultWriter().TaskID(),
			})
			return fmt.Errorf("process video: %w", err)
		}
	}

	h.normalizeThumbnailPath(ctx, id)

	if remote {
		if err := h.syncOutputsToStore(ctx, video); err != nil {
			h.logger.Error(ctx, "failed to upload transcoded outputs", err, map[string]interface{}{
				"video_id": payload.VideoID,
				"task_id":  task.ResultWriter().TaskID(),
			})
			return fmt.Errorf("upload outputs: %w", err)
		}
	}

	h.logger.Info(ctx, "video processing completed", map[string]interface{}{
		"video_id": payload.VideoID,
		"task_id":  task.ResultWriter().TaskID(),
	})

	return nil
}

// stageRawFile materializes the raw object at the local FilePath recorded at
// upload time, which is where ProcessVideo points ffmpeg. Already-present
// files are reused so a retry does not re-download gigabytes.
func (h *VideoProcessingHandler) stageRawFile(ctx context.Context, video *domain.Video) error {
	if _, err := os.Stat(video.FilePath); err == nil {
		return nil
	}

	key := storage.Key("raw", filepath.Base(video.FilePath))
	obj, err := h.store.Open(ctx, key)
	if err != nil {
		return fmt.Errorf("opening raw object %s: %w", key, err)
	}
	defer obj.Close()

	if err := os.MkdirAll(filepath.Dir(video.FilePath), 0o755); err != nil {
		return fmt.Errorf("creating staging directory: %w", err)
	}

	dest, err := os.Create(video.FilePath)
	if err != nil {
		return fmt.Errorf("creating staging file: %w", err)
	}
	if _, err := io.Copy(dest, obj); err != nil {
		dest.Close()
		os.Remove(video.FilePath)
		return fmt.Errorf("staging raw video: %w", err)
	}
	// Close before any Remove: Windows cannot delete an open file, and a
	// deferred Close would swallow the flush error.
	if err := dest.Close(); err != nil {
		os.Remove(video.FilePath)
		return fmt.Errorf("flushing staged raw video: %w", err)
	}
	return nil
}

// syncOutputsToStore uploads everything the transcoder produced, then removes
// the local working copies. Removal is best-effort: a leftover file wastes
// disk, but failing the task over it would re-run nothing useful.
func (h *VideoProcessingHandler) syncOutputsToStore(ctx context.Context, video *domain.Video) error {
	videoID := video.ID.String()

	outputDir := filepath.Join(h.storageCfg.TranscodedPath, videoID)
	if err := h.uploadDir(ctx, outputDir, storage.Key("transcoded", videoID)); err != nil {
		return err
	}

	thumbnail := filepath.Join(h.storageCfg.ThumbnailPath, videoID+".jpg")
	if _, err := os.Stat(thumbnail); err == nil {
		if err := h.uploadFile(ctx, thumbnail, storage.Key("thumbnails", videoID+".jpg")); err != nil {
			return err
		}
	}

	for _, remove := range []func() error{
		func() error { return os.RemoveAll(outputDir) },
		func() error { return os.Remove(thumbnail) },
		func() error { return os.Remove(video.FilePath) },
	} {
		if err := remove(); err != nil && !errors.Is(err, os.ErrNotExist) {
			h.logger.Warn(ctx, "could not remove local working copy", map[string]interface{}{
				"video_id": videoID,
				"error":    err.Error(),
			})
		}
	}

	return nil
}

func (h *VideoProcessingHandler) uploadDir(ctx context.Context, dir, keyPrefix string) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("relativizing %s: %w", path, err)
		}
		return h.uploadFile(ctx, path, storage.Key(keyPrefix, filepath.ToSlash(rel)))
	})
}

func (h *VideoProcessingHandler) uploadFile(ctx context.Context, path, key string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("statting %s: %w", path, err)
	}

	if err := h.store.Save(ctx, key, f, info.Size(), contentTypeForFile(path)); err != nil {
		return fmt.Errorf("uploading %s: %w", key, err)
	}
	return nil
}

// contentTypeForFile labels the transcoder's outputs for object storage. Local
// serving never consults it, but a browser reading straight from a public
// bucket does.
func contentTypeForFile(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".m3u8":
		return "application/vnd.apple.mpegurl"
	case ".ts":
		return "video/MP2T"
	case ".mp4":
		return "video/mp4"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

// normalizeThumbnailPath rewrites the thumbnail path as a forward-slash key.
// ProcessVideo records it with filepath.Join, so on Windows the database held
// "thumbnails\<id>.jpg" and the backslash leaked into every video JSON
// response. On Unix the path already uses "/" and nothing is written.
func (h *VideoProcessingHandler) normalizeThumbnailPath(ctx context.Context, id uuid.UUID) {
	video, err := h.videoRepo.GetByID(ctx, id)
	if err != nil {
		h.logger.Warn(ctx, "could not load video to normalize thumbnail path", map[string]interface{}{
			"video_id": id,
			"error":    err.Error(),
		})
		return
	}
	if video.Status != domain.VideoStatusReady || video.ThumbnailPath == nil {
		return
	}

	normalized := filepath.ToSlash(*video.ThumbnailPath)
	if normalized == *video.ThumbnailPath {
		return
	}

	if err := h.videoRepo.MarkAsReady(ctx, id, video.AvailableQualities, normalized); err != nil {
		h.logger.Error(ctx, "failed to normalize thumbnail path", err, map[string]interface{}{
			"video_id": id,
		})
	}
}
