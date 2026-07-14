// Package storage abstracts where video bytes live. The API and worker address
// files by key; whether a key resolves to a path under the upload directory or
// to an object in MinIO is decided once, in New, from configuration.
package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

// FileInfo is the object metadata a streaming response needs: Size for
// Content-Length and ModTime for Last-Modified / If-Modified-Since handling.
type FileInfo struct {
	Size    int64
	ModTime time.Time
}

// Store reads and writes video files by key.
//
// Keys are always forward-slash separated, on every platform. A key's first
// segment names the storage area ("raw", "transcoded", "thumbnails"), which is
// how the MinIO backend picks a bucket and the local backend honours the
// per-area directory overrides in StorageConfig. Build keys with Key, never
// filepath.Join: the OS separator is exactly how a backslash once leaked into
// thumbnail_path in JSON responses.
type Store interface {
	Save(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Open returns a ReadSeekCloser, not just a ReadCloser: the MP4 fallback
	// handler serves HTTP Range requests through http.ServeContent, which
	// needs to seek.
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Stat(ctx context.Context, key string) (FileInfo, error)
	Delete(ctx context.Context, key string) error
	// DeletePrefix removes every object under prefix; deleting a video has to
	// take its whole transcoded directory with it.
	DeletePrefix(ctx context.Context, prefix string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// New picks the backend from configuration: MinIO when enabled, the local
// filesystem otherwise.
func New(cfg *config.Config) (Store, error) {
	if cfg.MinIO.Enabled {
		return NewMinIO(cfg.MinIO)
	}
	return NewLocal(cfg.Storage), nil
}

// IsRemote reports whether s keeps objects somewhere other than the local
// filesystem. The transcoding worker uses this to decide whether finished
// renditions must be uploaded and the local working copies removed; with a
// local store the files are already in their final place and copying them
// onto themselves would truncate them.
func IsRemote(s Store) bool {
	_, local := s.(*Local)
	return !local
}

// Key joins segments into a storage key with forward slashes on every
// platform.
func Key(segments ...string) string {
	return path.Join(segments...)
}

// validateKey rejects keys that could name a file outside the store: absolute
// paths, Windows drive letters, backslashes, and ".." traversal. Both backends
// check before touching anything, so a hostile key is refused rather than
// resolved.
func validateKey(key string) error {
	if key == "" || strings.HasPrefix(key, "/") || strings.ContainsAny(key, `\:`) {
		return fmt.Errorf("%w: %q", domain.ErrStorageKeyInvalid, key)
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == ".." {
			return fmt.Errorf("%w: %q", domain.ErrStorageKeyInvalid, key)
		}
	}
	return nil
}
