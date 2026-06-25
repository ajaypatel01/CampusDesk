package storage

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Client struct {
	svc      *s3.S3
	uploader *s3manager.Uploader
	bucket   string
}

type Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

func New(cfg Config) (*Client, error) {
	awsCfg := &aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	}
	if cfg.Endpoint != "" {
		protocol := "https"
		if !cfg.UseSSL {
			protocol = "http"
		}
		awsCfg.Endpoint = aws.String(fmt.Sprintf("%s://%s", protocol, cfg.Endpoint))
		awsCfg.S3ForcePathStyle = aws.Bool(true) // required for MinIO
		awsCfg.DisableSSL = aws.Bool(!cfg.UseSSL)
	}

	sess, err := session.NewSession(awsCfg)
	if err != nil {
		return nil, fmt.Errorf("create s3 session: %w", err)
	}
	svc := s3.New(sess)
	return &Client{svc: svc, uploader: s3manager.NewUploader(sess), bucket: cfg.Bucket}, nil
}

func (c *Client) Enabled() bool {
	return c != nil && c.bucket != ""
}

// Upload stores data in S3 and returns the object key.
func (c *Client) Upload(key, contentType string, data []byte) error {
	_, err := c.uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("s3 upload %s: %w", key, err)
	}
	return nil
}

// Download fetches an object's bytes.
func (c *Client) Download(key string) ([]byte, error) {
	buf := aws.NewWriteAtBuffer([]byte{})
	dl := s3manager.NewDownloaderWithClient(c.svc)
	_, err := dl.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 download %s: %w", key, err)
	}
	return buf.Bytes(), nil
}

// PresignedURL generates a time-limited GET URL for an object.
func (c *Client) PresignedURL(key string, expiry time.Duration) (string, error) {
	req, _ := c.svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	url, err := req.Presign(expiry)
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return url, nil
}

// GetReader returns an io.ReadCloser for streaming downloads (for PDF image embedding).
func (c *Client) GetReader(key string) (io.ReadCloser, error) {
	out, err := c.svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get %s: %w", key, err)
	}
	return out.Body, nil
}
