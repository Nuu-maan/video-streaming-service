package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketRaw       string
	BucketProcessed string
	BucketThumbs    string
}

type MinIOStorage struct {
	client          *minio.Client
	bucketRaw       string
	bucketProcessed string
	bucketThumbs    string
}

func NewMinIOStorage(cfg MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	storage := &MinIOStorage{
		client:          client,
		bucketRaw:       cfg.BucketRaw,
		bucketProcessed: cfg.BucketProcessed,
		bucketThumbs:    cfg.BucketThumbs,
	}

	if err := storage.ensureBuckets(context.Background()); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *MinIOStorage) ensureBuckets(ctx context.Context) error {
	buckets := []string{s.bucketRaw, s.bucketProcessed, s.bucketThumbs}

	for _, bucket := range buckets {
		exists, err := s.client.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("failed to check bucket %s: %w", bucket, err)
		}

		if !exists {
			if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
	}

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

	for _, bucket := range []string{s.bucketProcessed, s.bucketThumbs} {
		policy := fmt.Sprintf(publicReadPolicy, bucket)
		if err := s.client.SetBucketPolicy(ctx, bucket, policy); err != nil {
			return fmt.Errorf("failed to set policy for bucket %s: %w", bucket, err)
		}
	}

	return nil
}

func (s *MinIOStorage) UploadFile(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (s *MinIOStorage) UploadRawVideo(ctx context.Context, key string, reader io.Reader, size int64) error {
	return s.UploadFile(ctx, s.bucketRaw, key, reader, size, "video/mp4")
}

func (s *MinIOStorage) UploadProcessedVideo(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return s.UploadFile(ctx, s.bucketProcessed, key, reader, size, contentType)
}

func (s *MinIOStorage) UploadThumbnail(ctx context.Context, key string, reader io.Reader, size int64) error {
	return s.UploadFile(ctx, s.bucketThumbs, key, reader, size, "image/jpeg")
}

func (s *MinIOStorage) DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from %s/%s: %w", bucket, key, err)
	}
	return obj, nil
}

func (s *MinIOStorage) DownloadRawVideo(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.DownloadFile(ctx, s.bucketRaw, key)
}

func (s *MinIOStorage) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s/%s: %w", bucket, key, err)
	}
	return presignedURL.String(), nil
}

func (s *MinIOStorage) GetProcessedVideoURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return s.GetPresignedURL(ctx, s.bucketProcessed, key, expiry)
}

func (s *MinIOStorage) GetThumbnailURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return s.GetPresignedURL(ctx, s.bucketThumbs, key, expiry)
}

func (s *MinIOStorage) GetPublicURL(bucket, key string) string {
	return fmt.Sprintf("/%s/%s", bucket, key)
}

func (s *MinIOStorage) DeleteFile(ctx context.Context, bucket, key string) error {
	err := s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (s *MinIOStorage) DeleteRawVideo(ctx context.Context, key string) error {
	return s.DeleteFile(ctx, s.bucketRaw, key)
}

func (s *MinIOStorage) DeleteProcessedVideo(ctx context.Context, key string) error {
	return s.DeleteFile(ctx, s.bucketProcessed, key)
}

func (s *MinIOStorage) DeleteThumbnail(ctx context.Context, key string) error {
	return s.DeleteFile(ctx, s.bucketThumbs, key)
}

func (s *MinIOStorage) ListFiles(ctx context.Context, bucket, prefix string) ([]string, error) {
	var files []string

	objectCh := s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		files = append(files, object.Key)
	}

	return files, nil
}

func (s *MinIOStorage) FileExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

func (s *MinIOStorage) GetFileInfo(ctx context.Context, bucket, key string) (*minio.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info %s/%s: %w", bucket, key, err)
	}
	return &info, nil
}

func (s *MinIOStorage) CopyFile(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	src := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcKey,
	}
	dst := minio.CopyDestOptions{
		Bucket: dstBucket,
		Object: dstKey,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy file from %s/%s to %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}
	return nil
}

func (s *MinIOStorage) BucketRaw() string {
	return s.bucketRaw
}

func (s *MinIOStorage) BucketProcessed() string {
	return s.bucketProcessed
}

func (s *MinIOStorage) BucketThumbs() string {
	return s.bucketThumbs
}

func (s *MinIOStorage) HealthCheck(ctx context.Context) error {
	_, err := s.client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("MinIO health check failed: %w", err)
	}
	return nil
}
