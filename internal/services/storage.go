package services

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"ethiopia-dating-app/internal/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageService struct {
	cfg         *config.Config
	s3Client    *s3.S3
	minioClient *minio.Client
	useMinIO    bool
}

func NewStorageService(cfg *config.Config) (*StorageService, error) {
	service := &StorageService{cfg: cfg}

	// Check if MinIO is configured
	if cfg.MinIOEndpoint != "" {
		service.useMinIO = true
		minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
			Secure: cfg.MinIOUseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create MinIO client: %w", err)
		}
		service.minioClient = minioClient
	} else {
		// Use AWS S3
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(cfg.AWSRegion),
			Credentials: credentials.NewStaticCredentials(
				cfg.AWSAccessKeyID,
				cfg.AWSSecretAccessKey,
				"",
			),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS session: %w", err)
		}
		service.s3Client = s3.New(sess)
	}

	return service, nil
}

func (s *StorageService) UploadFile(file io.Reader, filename, contentType string) (string, error) {
	if s.useMinIO {
		return s.uploadToMinIO(file, filename, contentType)
	}
	return s.uploadToS3(file, filename, contentType)
}

func (s *StorageService) DeleteFile(url string) error {
	// Extract key from URL
	key := s.extractKeyFromURL(url)
	if key == "" {
		return fmt.Errorf("invalid file URL")
	}

	if s.useMinIO {
		return s.deleteFromMinIO(key)
	}
	return s.deleteFromS3(key)
}

func (s *StorageService) uploadToS3(file io.Reader, filename, contentType string) (string, error) {
	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Upload to S3
	_, err = s.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Return public URL
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.cfg.S3Bucket, s.cfg.AWSRegion, filename)
	return url, nil
}

func (s *StorageService) uploadToMinIO(file io.Reader, filename, contentType string) (string, error) {
	// Upload to MinIO
	_, err := s.minioClient.PutObject(
		s.cfg.S3Bucket,
		filename,
		file,
		-1,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	// Return public URL
	protocol := "http"
	if s.cfg.MinIOUseSSL {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s/%s/%s", protocol, s.cfg.MinIOEndpoint, s.cfg.S3Bucket, filename)
	return url, nil
}

func (s *StorageService) deleteFromS3(key string) error {
	_, err := s.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	return nil
}

func (s *StorageService) deleteFromMinIO(key string) error {
	err := s.minioClient.RemoveObject(s.cfg.S3Bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete from MinIO: %w", err)
	}
	return nil
}

func (s *StorageService) extractKeyFromURL(url string) string {
	// Extract key from S3 URL
	if strings.Contains(url, "amazonaws.com") {
		parts := strings.Split(url, "/")
		if len(parts) > 3 {
			return strings.Join(parts[3:], "/")
		}
	}

	// Extract key from MinIO URL
	if strings.Contains(url, s.cfg.MinIOEndpoint) {
		parts := strings.Split(url, "/")
		if len(parts) > 3 {
			return strings.Join(parts[3:], "/")
		}
	}

	return ""
}

func (s *StorageService) GeneratePresignedURL(filename string, expiration time.Duration) (string, error) {
	if s.useMinIO {
		return s.generateMinIOPresignedURL(filename, expiration)
	}
	return s.generateS3PresignedURL(filename, expiration)
}

func (s *StorageService) generateS3PresignedURL(filename string, expiration time.Duration) (string, error) {
	req, _ := s.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(filename),
	})

	url, err := req.Presign(expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

func (s *StorageService) generateMinIOPresignedURL(filename string, expiration time.Duration) (string, error) {
	url, err := s.minioClient.PresignedGetObject(s.cfg.S3Bucket, filename, expiration, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

func (s *StorageService) CreateBucket() error {
	if s.useMinIO {
		return s.createMinIOBucket()
	}
	return s.createS3Bucket()
}

func (s *StorageService) createS3Bucket() error {
	_, err := s.s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(s.cfg.S3Bucket),
	})
	if err != nil {
		// Check if bucket already exists
		if !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			return fmt.Errorf("failed to create S3 bucket: %w", err)
		}
	}
	return nil
}

func (s *StorageService) createMinIOBucket() error {
	exists, err := s.minioClient.BucketExists(s.cfg.S3Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.minioClient.MakeBucket(s.cfg.S3Bucket, "")
		if err != nil {
			return fmt.Errorf("failed to create MinIO bucket: %w", err)
		}
	}
	return nil
}

// Helper function to generate unique filename
func GenerateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%d_%s%s", timestamp, generateRandomString(8), ext)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
