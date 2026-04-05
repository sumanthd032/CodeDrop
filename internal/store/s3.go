package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Store struct {
	client *s3.Client
	bucket string
}

// NewS3Store initializes the S3 client.
// - If S3_ENDPOINT is set, it points to a custom endpoint (MinIO for local dev).
// - Otherwise, it uses the standard AWS credential chain (env vars, IAM role, etc.)
func NewS3Store() (*Store, error) {
	bucket := getEnv("S3_BUCKET", "codedrop-bucket")
	region := getEnv("AWS_REGION", "us-east-1")
	endpoint := os.Getenv("S3_ENDPOINT")

	var cfg aws.Config
	var err error

	if endpoint != "" {
		// Local dev mode: custom endpoint (MinIO) with static credentials
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, reg string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: region,
			}, nil
		})
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithEndpointResolverWithOptions(resolver),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				getEnv("AWS_ACCESS_KEY_ID", "minioadmin"),
				getEnv("AWS_SECRET_ACCESS_KEY", "minioadmin"),
				"",
			)),
			config.WithRegion(region),
		)
	} else {
		// Production mode: standard AWS credential chain (AWS_ACCESS_KEY_ID / IAM role)
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.UsePathStyle = true // Required for MinIO
		}
	})

	return &Store{client: client, bucket: bucket}, nil
}

// UploadChunk saves a piece of the file
func (s *Store) UploadChunk(key string, data []byte) error {
	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload chunk %s: %w", key, err)
	}
	return nil
}

// DownloadChunk retrieves a piece of the file
func (s *Store) DownloadChunk(key string) ([]byte, error) {
	resp, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download chunk %s: %w", key, err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// DeleteChunk removes a piece of the file (used for cleanup/expiry)
func (s *Store) DeleteChunk(key string) error {
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
