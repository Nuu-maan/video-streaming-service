package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
	"github.com/Nuu-maan/video-streaming-service/internal/repository"
	"github.com/Nuu-maan/video-streaming-service/internal/storage"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
	"github.com/Nuu-maan/video-streaming-service/pkg/validator"
)

// UploadService persists an uploaded video to storage and records it in the
// repository. Transcoding is not its concern: it leaves the video in
// VideoStatusUploading for the worker to pick up.
type UploadService struct {
	videoRepo     repository.VideoRepository
	ffmpegService *FFmpegService
	storageCfg    *config.StorageConfig
	store         storage.Store
	log           *logger.Logger
}

func NewUploadService(
	videoRepo repository.VideoRepository,
	ffmpegService *FFmpegService,
	storageCfg *config.StorageConfig,
	store storage.Store,
	log *logger.Logger,
) *UploadService {
	return &UploadService{
		videoRepo:     videoRepo,
		ffmpegService: ffmpegService,
		storageCfg:    storageCfg,
		store:         store,
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

// UploadVideo validates the request, streams the file into storage, probes it
// for metadata, and records it. If anything fails after the file lands in
// storage, the file is removed: a half-written upload with no database row is
// a leak.
func (s *UploadService) UploadVideo(ctx context.Context, req UploadRequest) (video *domain.Video, err error) {
	title := validator.SanitizeString(req.Title)
	description := validator.SanitizeString(req.Description)

	if err := validator.ValidateTitle(title); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidTitle, err)
	}
	if err := validator.ValidateDescription(description); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}
	if err := validator.ValidateVideoFile(req.File, req.Header, s.storageCfg.MaxFileSize); err != nil {
		return nil, err
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = domain.VisibilityPublic
	}
	if !visibility.IsValid() {
		return nil, fmt.Errorf("%w: unknown visibility %q", domain.ErrInvalidInput, visibility)
	}

	key, filePath, err := s.persistFile(ctx, req.File, req.Header)
	if err != nil {
		return nil, err
	}
	// Any failure past this point must not leave an orphaned file behind.
	defer func() {
		if err != nil {
			if removeErr := s.store.Delete(ctx, key); removeErr != nil {
				s.log.Error(ctx, "failed to clean up orphaned upload", removeErr, map[string]interface{}{
					"key": key,
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

	// ffprobe wants a local path, which only exists when the store is local.
	// A remote upload goes unprobed here; the transcoding worker probes again
	// once it has staged the file, and fails the video properly if it is
	// truly corrupt.
	if !storage.IsRemote(s.store) {
		if metadata, probeErr := s.ffmpegService.ExtractMetadata(ctx, filePath); probeErr != nil {
			s.log.Warn(ctx, "could not probe uploaded video; storing without metadata", map[string]interface{}{
				"path":  filePath,
				"error": probeErr.Error(),
			})
		} else {
			video.Duration = int(metadata.Duration)
			video.OriginalResolution = fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
		}
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

// persistFile streams the upload into the store under a generated name and
// returns its key along with the video's FilePath. The stored name is derived
// from a fresh UUID, never from user input, so a crafted filename cannot
// escape the upload area.
//
// FilePath is where the transcoding worker expects a local working copy of the
// raw file: the local store writes it there directly, and when the store is
// remote the worker re-materializes it from the raw object before running
// ffmpeg.
func (s *UploadService) persistFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (key, filePath string, err error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	name := uuid.New().String() + ext

	key = storage.Key("raw", name)
	if err := s.store.Save(ctx, key, file, header.Size, mimeTypeOf(header)); err != nil {
		return "", "", fmt.Errorf("storing upload: %w", err)
	}

	return key, filepath.Join(s.storageCfg.UploadPath, "raw", name), nil
}

// RemoveVideoFiles deletes everything storage holds for a video: the raw
// upload, the transcoded directory, and the thumbnail. It belongs beside every
// hard delete of a videos row — without it the files sit in storage forever,
// still fetchable through the static /uploads mount or the public MinIO
// buckets. It is best-effort by design: callers run it after the row is gone,
// when failing their request would only report a delete that did happen as one
// that did not, so failures are logged for manual reaping instead.
//
// Keys are rebuilt the same way their writers built them (persistFile for raw,
// the queue worker for the rest), and both backends treat deleting an absent
// key as a no-op, so a video that never finished transcoding cleans up fine.
func (s *UploadService) RemoveVideoFiles(ctx context.Context, video *domain.Video) {
	report := func(err error, key string) {
		if err != nil {
			s.log.Error(ctx, "video deleted but storage cleanup failed", err, map[string]interface{}{
				"video_id": video.ID,
				"key":      key,
			})
		}
	}

	if video.FilePath != "" {
		rawKey := storage.Key("raw", filepath.Base(video.FilePath))
		report(s.store.Delete(ctx, rawKey), rawKey)
	}

	transcodedPrefix := storage.Key("transcoded", video.ID.String())
	report(s.store.DeletePrefix(ctx, transcodedPrefix), transcodedPrefix)

	thumbnailKey := storage.Key("thumbnails", video.ID.String()+".jpg")
	report(s.store.Delete(ctx, thumbnailKey), thumbnailKey)
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
