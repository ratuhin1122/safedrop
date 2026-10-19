package storage

import (
	"context"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type S3Config struct {
	Endpoint, Region, Bucket                   string
	AccessKeyID, SecretAccessKey, SessionToken string
	ForcePathStyle                             bool
}

type S3 struct {
	client    *s3.Client
	uploader  *manager.Uploader
	presigner *s3.PresignClient
	bucket    string
}

var _ Store = (*S3)(nil)

func NewS3(cfg S3Config) (*S3, error) {
	if cfg.Bucket == "" || cfg.Region == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("S3 bucket, region, and credentials are required")
	}
	if cfg.Endpoint != "" {
		u, err := url.Parse(cfg.Endpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("invalid S3 endpoint")
		}
	}
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken),
		BaseEndpoint: optionalString(cfg.Endpoint), UsePathStyle: cfg.ForcePathStyle,
		// Custom S3 endpoints need not implement optional AWS checksum trailers.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 5 << 20
		u.Concurrency = 1
		u.LeavePartsOnError = false
	})
	return &S3{client, uploader, s3.NewPresignClient(client), cfg.Bucket}, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return aws.String(value)
}

func (s *S3) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if body == nil {
		return errors.New("upload reader is required")
	}
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: contextReader{ctx, body},
		ContentType: optionalString(contentType),
	})
	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, s3Error(err)
	}
	return object.Body, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	err = s3Error(err)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (s *S3) Move(ctx context.Context, source, target string) error {
	if err := validateKey(source); err != nil {
		return err
	}
	if err := validateKey(target); err != nil {
		return err
	}
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(source)})
	if err != nil {
		return s3Error(err)
	}
	defer object.Body.Close()
	if source == target {
		return nil
	}
	// Stream through the multipart uploader, avoiding CopyObject's 5 GiB limit.
	if err := s.Put(ctx, target, object.Body, aws.ToString(object.ContentType)); err != nil {
		return err
	}
	return s.Delete(ctx, source)
}

func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if err := validateDownload(key, ttl); err != nil {
		return "", err
	}
	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
		ResponseContentDisposition: aws.String("attachment"),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func s3Error(err error) error {
	var api smithy.APIError
	if errors.As(err, &api) && (api.ErrorCode() == "NoSuchKey" || api.ErrorCode() == "NotFound") {
		return errors.Join(ErrNotFound, err)
	}
	return err
}
