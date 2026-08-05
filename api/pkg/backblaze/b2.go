package backblaze

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// B2Client is an S3-compatible client for Backblaze B2 Gen2.
type B2Client struct {
	client     *s3.Client
	bucketName string
}

// NewB2Client creates a new Backblaze B2 client using S3-compatible API.
func NewB2Client(ctx context.Context, accountID, applicationKey, bucketName, endpoint string) (*B2Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-west-002"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accountID, applicationKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &B2Client{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadFile uploads a file to B2 and returns its public URL.
func (c *B2Client) UploadFile(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("upload to b2: %w", err)
	}

	return fmt.Sprintf("https://%s.s3.us-west-002.backblazeb2.com/%s", c.bucketName, key), nil
}

// GeneratePresignedURL creates a pre-signed URL for downloading a file.
// Expires after the given duration (default 1 hour for sensitive documents).
func (c *B2Client) GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.client)
	presigned, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("generate presigned url: %w", err)
	}
	return presigned.URL, nil
}

// GeneratePresignedUploadURL creates a pre-signed URL for uploading a file.
func (c *B2Client) GeneratePresignedUploadURL(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.client)
	presigned, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("generate presigned upload url: %w", err)
	}
	return presigned.URL, nil
}

// DeleteFile removes a file from B2.
func (c *B2Client) DeleteFile(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete from b2: %w", err)
	}
	return nil
}
