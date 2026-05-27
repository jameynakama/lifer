package r2

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

// Client uploads objects to a Cloudflare R2 bucket via the S3-compatible API.
type Client struct {
	s3     *s3.Client
	bucket string
	pubURL string // e.g. "https://pub-abc.r2.dev" -- no trailing slash
}

// New creates a Client for a Cloudflare R2 bucket.
// accountID is the Cloudflare account ID (used to build the endpoint URL).
// pubURL is the public base URL served to browsers, e.g. "https://pub-abc.r2.dev".
func New(accountID, accessKeyID, secretKey, bucket, pubURL string) (*Client, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	return NewWithEndpoint(endpoint, accessKeyID, secretKey, bucket, pubURL)
}

// NewWithEndpoint creates a Client with a custom S3 endpoint URL.
// Use this in tests to point at an httptest.Server.
func NewWithEndpoint(endpoint, accessKeyID, secretKey, bucket, pubURL string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("r2: load config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return &Client{s3: client, bucket: bucket, pubURL: pubURL}, nil
}

// Upload reads body, uploads it to R2 at key, and returns the full public URL.
// Body is buffered in memory (files are small -- a few MB each).
func (c *Client) Upload(ctx context.Context, key, contentType string, body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("r2: read body for %s: %w", key, err)
	}
	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("r2: put %s: %w", key, err)
	}
	return c.pubURL + "/" + key, nil
}
