package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

// newTestLocal roots a Local store in a fresh temp directory, with the
// transcoded and thumbnail areas mounted the way storage.New wires them from
// StorageConfig.
func newTestLocal(t *testing.T) (*Local, string) {
	t.Helper()
	root := t.TempDir()
	store := NewLocal(config.StorageConfig{
		UploadPath:     root,
		TranscodedPath: filepath.Join(root, "transcoded"),
		ThumbnailPath:  filepath.Join(root, "thumbnails"),
	})
	return store, root
}

func TestLocalRejectsHostileKeys(t *testing.T) {
	store, root := newTestLocal(t)
	ctx := context.Background()

	keys := []string{
		"",
		"..",
		"../outside.txt",
		"raw/../../outside.txt",
		"/etc/passwd",
		`raw\..\..\outside.txt`,
		`C:/Windows/system32/drivers/etc/hosts`,
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			if err := store.Save(ctx, key, strings.NewReader("x"), 1, "text/plain"); !errors.Is(err, domain.ErrStorageKeyInvalid) {
				t.Fatalf("Save(%q) = %v, want ErrStorageKeyInvalid", key, err)
			}
			if _, err := store.Open(ctx, key); !errors.Is(err, domain.ErrStorageKeyInvalid) {
				t.Fatalf("Open(%q) = %v, want ErrStorageKeyInvalid", key, err)
			}
		})
	}

	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("hostile keys touched the filesystem: entries=%v err=%v", entries, err)
	}
}

func TestLocalRoundTrip(t *testing.T) {
	store, root := newTestLocal(t)
	ctx := context.Background()

	key := Key("raw", "video.mp4")
	body := "not really a video"

	if err := store.Save(ctx, key, strings.NewReader(body), int64(len(body)), "video/mp4"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The raw area must land exactly where the pre-Store code wrote it.
	if _, err := os.Stat(filepath.Join(root, "raw", "video.mp4")); err != nil {
		t.Fatalf("saved file not at expected path: %v", err)
	}

	obj, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := io.ReadAll(obj)
	obj.Close()
	if err != nil || string(got) != body {
		t.Fatalf("read %q (err=%v), want %q", got, err, body)
	}

	info, err := store.Stat(ctx, key)
	if err != nil || info.Size != int64(len(body)) {
		t.Fatalf("Stat = %+v (err=%v), want size %d", info, err, len(body))
	}

	exists, err := store.Exists(ctx, key)
	if err != nil || !exists {
		t.Fatalf("Exists = %v (err=%v), want true", exists, err)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Deleting again must stay silent so cleanup paths can run unconditionally.
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("repeated Delete: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, domain.ErrStorageObjectNotFound) {
		t.Fatalf("Open after delete = %v, want ErrStorageObjectNotFound", err)
	}
}

func TestLocalMountsFollowConfiguredDirectories(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	store := NewLocal(config.StorageConfig{
		UploadPath:     root,
		TranscodedPath: filepath.Join(elsewhere, "transcoded"),
		ThumbnailPath:  filepath.Join(elsewhere, "thumbs"),
	})
	ctx := context.Background()

	key := Key("transcoded", "vid-1", "hls", "master.m3u8")
	if err := store.Save(ctx, key, strings.NewReader("#EXTM3U"), 7, "application/vnd.apple.mpegurl"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	want := filepath.Join(elsewhere, "transcoded", "vid-1", "hls", "master.m3u8")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("transcoded key did not resolve through TranscodedPath: %v", err)
	}

	if err := store.DeletePrefix(ctx, Key("transcoded", "vid-1")); err != nil {
		t.Fatalf("DeletePrefix: %v", err)
	}
	if _, err := os.Stat(filepath.Join(elsewhere, "transcoded", "vid-1")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("DeletePrefix left the directory behind: %v", err)
	}
}

func TestLocalSaveRemovesShortWrite(t *testing.T) {
	store, root := newTestLocal(t)
	ctx := context.Background()

	key := Key("raw", "truncated.mp4")
	if err := store.Save(ctx, key, strings.NewReader("abc"), 10, "video/mp4"); err == nil {
		t.Fatal("Save with short reader succeeded, want error")
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "truncated.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial file left behind: %v", err)
	}
}
