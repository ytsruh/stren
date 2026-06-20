// Package utils: storage.go wraps the AWS S3 client used to talk to R2
// (S3-compatible). It provides presigned-URL generation for direct browser
// uploads and helpers to delete objects.
package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageConfig holds the R2 (or any S3-compatible) credentials and bucket info.
// Fields are populated from environment variables and validated on startup.
type StorageConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Bucket     string
	PublicURL  string // Public base URL used to build display URLs (e.g., https://pub-xxx.r2.dev)
}

// LoadStorageConfig reads the storage environment variables and returns a
// validated config. It returns an error if any required variable is missing.
func LoadStorageConfig() (*StorageConfig, error) {
	sc := &StorageConfig{
		Endpoint:  os.Getenv("STORAGE_ENDPOINT"),
		AccessKey: os.Getenv("STORAGE_ACCESS_KEY"),
		SecretKey: os.Getenv("STORAGE_SECRET_KEY"),
		Bucket:    os.Getenv("STORAGE_BUCKET"),
		PublicURL: os.Getenv("STORAGE_PUBLIC_URL"),
	}
	if sc.Endpoint == "" || sc.AccessKey == "" || sc.SecretKey == "" || sc.Bucket == "" {
		return nil, fmt.Errorf("missing required storage environment variables (STORAGE_ENDPOINT, STORAGE_ACCESS_KEY, STORAGE_SECRET_KEY, STORAGE_BUCKET)")
	}
	storageConfig = sc
	return sc, nil
}

// storageConfig holds the loaded config. Set by LoadStorageConfig.
var storageConfig *StorageConfig

// s3Client is a lazily-initialized S3 client.
var s3Client *s3.Client

// GetS3Client returns the package-level S3 client, creating it on first use.
// It returns an error if LoadStorageConfig has not been called successfully.
func GetS3Client() (*s3.Client, error) {
	if s3Client != nil {
		return s3Client, nil
	}
	sc := storageConfig
	if sc == nil {
		return nil, fmt.Errorf("storage config not loaded")
	}
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			sc.AccessKey, sc.SecretKey, "",
		)),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(sc.Endpoint)
		o.UsePathStyle = true
	})
	return s3Client, nil
}

// CreatePresignedPutURL generates a presigned URL for a browser PUT upload
// directly to R2. The URL is valid for 1 hour.
func CreatePresignedPutURL(key, contentType string) (string, error) {
	client, err := GetS3Client()
	if err != nil {
		return "", err
	}
	sc := storageConfig
	presignClient := s3.NewPresignClient(client)
	presignedReq, err := presignClient.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(sc.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(po *s3.PresignOptions) {
		po.Expires = time.Hour
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedReq.URL, nil
}

// CreatePresignedGetURL generates a presigned GET URL for a private object.
// (Not used in the current public-bucket setup, but kept for parity.)
func CreatePresignedGetURL(key string) (string, error) {
	client, err := GetS3Client()
	if err != nil {
		return "", err
	}
	sc := storageConfig
	presignClient := s3.NewPresignClient(client)
	presignedReq, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(sc.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedReq.URL, nil
}

// DeleteObject removes an object from R2 by key. Errors are wrapped.
func DeleteObject(key string) error {
	client, err := GetS3Client()
	if err != nil {
		return err
	}
	sc := storageConfig
	_, err = client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(sc.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GetObject fetches the raw bytes stream of an R2 object by key. The
// caller is responsible for closing the returned io.ReadCloser. Errors
// (including "not found" from R2) are wrapped.
func GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	client, err := GetS3Client()
	if err != nil {
		return nil, err
	}
	sc := storageConfig
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sc.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %q: %w", key, err)
	}
	return out.Body, nil
}

// PublicURLFor returns the public URL for an object key, given the configured
// STORAGE_PUBLIC_URL base. Returns "" if key is empty.
func PublicURLFor(key string) string {
	if key == "" || storageConfig == nil {
		return ""
	}
	return storageConfig.PublicURL + "/" + key
}
