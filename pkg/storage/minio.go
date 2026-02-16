package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage defines the interface for file storage operations
type Storage interface {
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, folder string) (*UploadResult, error)
	Delete(ctx context.Context, objectName string) error
	GetPublicURL(objectName string) string
}

// UploadResult contains the result of a file upload
type UploadResult struct {
	URL      string
	Key      string // object key in storage
	FileName string
	FileSize int64
	MimeType string
}

// MinIOStorage implements Storage interface using MinIO
type MinIOStorage struct {
	client    *minio.Client
	bucket    string
	endpoint  string
	publicURL string // External URL
	useSSL    bool
}

// Config holds MinIO connection configuration
type Config struct {
	Endpoint  string
	PublicURL string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// NewMinIO creates a new MinIO storage client
func NewMinIO(cfg Config) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MinIO: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("📦 Created MinIO bucket: %s", cfg.Bucket)

		// Set bucket policy to public read
		policy := `{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::` + cfg.Bucket + `/*"]
			}]
		}`
		if err := client.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
			log.Printf("⚠️  Failed to set bucket policy: %v", err)
		}
	}

	return &MinIOStorage{
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		publicURL: cfg.PublicURL,
		useSSL:    cfg.UseSSL,
	}, nil
}

// Upload uploads a file to MinIO
func (s *MinIOStorage) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, folder string) (*UploadResult, error) {
	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	uniqueName := fmt.Sprintf("%s/%s/%s%s",
		folder,
		time.Now().Format("2006/01/02"),
		uuid.New().String(),
		ext,
	)

	// Detect content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(ext)
	}

	// Upload to MinIO
	_, err := s.client.PutObject(ctx, s.bucket, uniqueName, file, header.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &UploadResult{
		URL:      s.GetPublicURL(uniqueName),
		Key:      uniqueName,
		FileName: header.Filename,
		FileSize: header.Size,
		MimeType: contentType,
	}, nil
}

// Delete removes a file from MinIO
func (s *MinIOStorage) Delete(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}

// GetPublicURL returns the public URL for an object
func (s *MinIOStorage) GetPublicURL(objectName string) string {
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.publicURL, "/"), s.bucket, objectName)
	}

	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, s.bucket, objectName)
}

// UploadFromReader uploads from an io.Reader (useful for internal operations)
func (s *MinIOStorage) UploadFromReader(ctx context.Context, reader io.Reader, size int64, objectName, contentType string) (*UploadResult, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &UploadResult{
		URL:      s.GetPublicURL(objectName),
		Key:      objectName,
		MimeType: contentType,
	}, nil
}

// detectContentType returns MIME type based on file extension
func detectContentType(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	case ".pdf":
		return "application/pdf"
	case ".doc", ".docx":
		return "application/msword"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
