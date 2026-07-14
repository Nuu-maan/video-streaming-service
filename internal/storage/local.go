package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

// Local is the filesystem Store, rooted at the upload directory. The
// "transcoded" and "thumbnails" areas resolve through their own configured
// directories rather than under the root, because STORAGE_TRANSCODED_PATH and
// STORAGE_THUMBNAIL_PATH are independently overridable and the transcoding
// service (which writes those files directly, for ffmpeg) honours them.
// Resolving both sides from the same config values is what keeps the writer
// and the reader pointed at the same directory.
type Local struct {
	root   string
	mounts map[string]string
}

func NewLocal(cfg config.StorageConfig) *Local {
	return &Local{
		root: cfg.UploadPath,
		mounts: map[string]string{
			"transcoded": cfg.TranscodedPath,
			"thumbnails": cfg.ThumbnailPath,
		},
	}
}

// resolve maps a key onto a filesystem path. validateKey has already refused
// anything that could climb out of the mapped directories.
func (l *Local) resolve(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	area, rest, _ := strings.Cut(key, "/")
	if dir, ok := l.mounts[area]; ok {
		return filepath.Join(dir, filepath.FromSlash(rest)), nil
	}
	return filepath.Join(l.root, filepath.FromSlash(key)), nil
}

// Save streams r to the key's path. contentType is unused: a filesystem has no
// content-type, and serving derives it from the extension.
func (l *Local) Save(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	path, err := l.resolve(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", key, err)
	}

	dest, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", key, err)
	}

	written, err := io.Copy(dest, r)
	if err != nil {
		dest.Close()
		os.Remove(path)
		return fmt.Errorf("writing %s: %w", key, err)
	}

	// Close before any Remove: on Windows the file cannot be removed while
	// open, and a deferred Close would also swallow the flush error.
	if err := dest.Close(); err != nil {
		os.Remove(path)
		return fmt.Errorf("flushing %s: %w", key, err)
	}

	if size >= 0 && written != size {
		os.Remove(path)
		return fmt.Errorf("writing %s: expected %d bytes, wrote %d", key, size, written)
	}

	return nil
}

func (l *Local) Open(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	path, err := l.resolve(key)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", domain.ErrStorageObjectNotFound, key)
		}
		return nil, fmt.Errorf("opening %s: %w", key, err)
	}
	return f, nil
}

func (l *Local) Stat(ctx context.Context, key string) (FileInfo, error) {
	path, err := l.resolve(key)
	if err != nil {
		return FileInfo{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FileInfo{}, fmt.Errorf("%w: %s", domain.ErrStorageObjectNotFound, key)
		}
		return FileInfo{}, fmt.Errorf("statting %s: %w", key, err)
	}
	return FileInfo{Size: info.Size(), ModTime: info.ModTime()}, nil
}

// Delete is idempotent: removing a key that is already gone is not an error,
// so cleanup paths can run unconditionally.
func (l *Local) Delete(ctx context.Context, key string) error {
	path, err := l.resolve(key)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("deleting %s: %w", key, err)
	}
	return nil
}

func (l *Local) DeletePrefix(ctx context.Context, prefix string) error {
	path, err := l.resolve(prefix)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("deleting prefix %s: %w", prefix, err)
	}
	return nil
}

func (l *Local) Exists(ctx context.Context, key string) (bool, error) {
	path, err := l.resolve(key)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("statting %s: %w", key, err)
	}
	return true, nil
}
