package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/internal/domain"
)

// MinIO is the object-store Store. A key's first segment selects the bucket —
// "raw/x.mp4" lands in the raw bucket as object "x.mp4" — so the bucket layout
// stays the three-bucket one the config describes, with the processed and
// thumbnail buckets world-readable for players and <img> tags.
type MinIO struct {
	client  *minio.Client
	buckets map[string]string
}

func NewMinIO(cfg config.MinIOConfig) (*MinIO, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating MinIO client: %w", err)
	}

	store := &MinIO{
		client: client,
		buckets: map[string]string{
			"raw":        cfg.BucketRaw,
			"transcoded": cfg.BucketProcessed,
			"thumbnails": cfg.BucketThumbs,
		},
	}

	if err := store.ensureBuckets(context.Background(), cfg); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *MinIO) ensureBuckets(ctx context.Context, cfg config.MinIOConfig) error {
	for _, bucket := range []string{cfg.BucketRaw, cfg.BucketProcessed, cfg.BucketThumbs} {
		exists, err := s.client.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("checking bucket %s: %w", bucket, err)
		}
		if !exists {
			if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("creating bucket %s: %w", bucket, err)
			}
		}
	}

	// The raw bucket is deliberately not listed: originals are reachable only
	// through the API, never by URL.
	publicReadPolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`

	for _, bucket := range []string{cfg.BucketProcessed, cfg.BucketThumbs} {
		policy := fmt.Sprintf(publicReadPolicy, bucket)
		if err := s.client.SetBucketPolicy(ctx, bucket, policy); err != nil {
			return fmt.Errorf("setting policy for bucket %s: %w", bucket, err)
		}
	}

	return nil
}

// locate maps a key onto its bucket and object name. A key whose area has no
// bucket is a programming error and is refused rather than guessed at.
func (s *MinIO) locate(key string) (bucket, object string, err error) {
	if err := validateKey(key); err != nil {
		return "", "", err
	}
	area, rest, _ := strings.Cut(key, "/")
	bucket, ok := s.buckets[area]
	if !ok || rest == "" {
		return "", "", fmt.Errorf("%w: no bucket for %q", domain.ErrStorageKeyInvalid, key)
	}
	return bucket, rest, nil
}

func (s *MinIO) Save(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	bucket, object, err := s.locate(key)
	if err != nil {
		return err
	}

	if _, err := s.client.PutObject(ctx, bucket, object, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return fmt.Errorf("uploading %s: %w", key, err)
	}
	return nil
}

func (s *MinIO) Open(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	bucket, object, err := s.locate(key)
	if err != nil {
		return nil, err
	}

	obj, err := s.client.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", key, err)
	}

	// GetObject defers the request until the first Read, so a missing object
	// would otherwise surface only mid-response. Stat forces the round trip
	// now, where the caller can still answer with a 404.
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if isNoSuchKey(err) {
			return nil, fmt.Errorf("%w: %s", domain.ErrStorageObjectNotFound, key)
		}
		return nil, fmt.Errorf("opening %s: %w", key, err)
	}

	return obj, nil
}

func (s *MinIO) Stat(ctx context.Context, key string) (FileInfo, error) {
	bucket, object, err := s.locate(key)
	if err != nil {
		return FileInfo{}, err
	}

	info, err := s.client.StatObject(ctx, bucket, object, minio.StatObjectOptions{})
	if err != nil {
		if isNoSuchKey(err) {
			return FileInfo{}, fmt.Errorf("%w: %s", domain.ErrStorageObjectNotFound, key)
		}
		return FileInfo{}, fmt.Errorf("statting %s: %w", key, err)
	}
	return FileInfo{Size: info.Size, ModTime: info.LastModified}, nil
}

func (s *MinIO) Delete(ctx context.Context, key string) error {
	bucket, object, err := s.locate(key)
	if err != nil {
		return err
	}

	if err := s.client.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting %s: %w", key, err)
	}
	return nil
}

func (s *MinIO) DeletePrefix(ctx context.Context, prefix string) error {
	bucket, object, err := s.locate(prefix)
	if err != nil {
		return err
	}

	// The trailing slash keeps "transcoded/<id>" from also matching another
	// object whose name merely starts with <id>.
	if !strings.HasSuffix(object, "/") {
		object += "/"
	}

	objects := s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    object,
		Recursive: true,
	})
	for obj := range objects {
		if obj.Err != nil {
			return fmt.Errorf("listing prefix %s: %w", prefix, obj.Err)
		}
		if err := s.client.RemoveObject(ctx, bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("deleting %s/%s: %w", bucket, obj.Key, err)
		}
	}
	return nil
}

func (s *MinIO) Exists(ctx context.Context, key string) (bool, error) {
	bucket, object, err := s.locate(key)
	if err != nil {
		return false, err
	}

	if _, err := s.client.StatObject(ctx, bucket, object, minio.StatObjectOptions{}); err != nil {
		if isNoSuchKey(err) {
			return false, nil
		}
		return false, fmt.Errorf("statting %s: %w", key, err)
	}
	return true, nil
}

func isNoSuchKey(err error) bool {
	return minio.ToErrorResponse(err).Code == "NoSuchKey"
}
