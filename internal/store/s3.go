package store

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Store struct {
	client *s3.Client
	bucket string
}

// NewS3Store initializes the S3 client pointed at MinIO (local) or AWS (prod)
func NewS3Store() (*Store, error) {
	// 1. Load configuration manually to support MinIO
	// In a real AWS environment, we wouldn't need to hardcode endpoints like this.
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           "http://127.0.0.1:9000", // Point to local MinIO
			SigningRegion: "us-east-1",            // MinIO defaults to this region
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for MinIO
	})

	return &Store{
		client: client,
		bucket: "codedrop-bucket", // Matches the bucket created in docker-compose
	}, nil
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
		return nil, fmt.Errorf("failed to download chunk %s: %w", err)
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