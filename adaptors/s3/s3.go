package s3

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/px4n/finspect/adaptors/cloud"
)

// Register registers the S3 provider
func Register() {
	cloud.RegisterProvider(cloud.ProviderS3, New)
}

// Adaptor implements the CloudAdaptor interface for AWS S3
type Adaptor struct {
	*cloud.BaseCloudAdaptor
	client *s3.Client
	bucket string
}

// New creates a new S3 adaptor
func New() cloud.Adaptor {
	return &Adaptor{
		BaseCloudAdaptor: cloud.NewBaseCloudAdaptor(cloud.ProviderS3),
	}
}

// Connect establishes connection to S3
func (a *Adaptor) Connect(ctx context.Context, cfg cloud.AdaptorConfig) error {
	// Build AWS config
	awsCfg, err := a.buildAWSConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("build AWS config: %w", err)
	}

	// Create S3 client
	a.client = s3.NewFromConfig(awsCfg)
	a.bucket = cfg.Bucket
	a.SetBucket(cfg.Bucket)

	// Test connection by listing buckets or checking bucket exists
	if a.bucket != "" {
		_, err = a.client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(a.bucket),
		})
		if err != nil {
			return fmt.Errorf("bucket %s not accessible: %w", a.bucket, err)
		}
	}

	return nil
}

// Disconnect closes the connection
func (a *Adaptor) Disconnect(_ context.Context) error {
	a.client = nil
	return nil
}

// Stat returns information about a file or directory
func (a *Adaptor) Stat(ctx context.Context, path string) (*cloud.FileInfo, error) {
	// Resolve path
	key := a.ResolvePath(path)

	// Check if it's a file
	headResp, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})

	if err == nil {
		// It's a file
		return &cloud.FileInfo{
			Path:         path,
			Size:         *headResp.ContentLength,
			ModTime:      *headResp.LastModified,
			IsDir:        false,
			ETag:         strings.Trim(*headResp.ETag, "\""),
			ContentType:  aws.ToString(headResp.ContentType),
			StorageClass: string(headResp.StorageClass),
			Metadata:     headResp.Metadata,
		}, nil
	}

	// Check if it's a directory by listing with prefix
	listResp, err := a.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(a.bucket),
		Prefix:    aws.String(key + "/"),
		MaxKeys:   aws.Int32(1),
		Delimiter: aws.String("/"),
	})

	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if len(listResp.Contents) > 0 || len(listResp.CommonPrefixes) > 0 {
		// It's a directory
		return &cloud.FileInfo{
			Path:    path,
			IsDir:   true,
			ModTime: time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("path not found: %s", path)
}

// List returns a list of files in a directory
func (a *Adaptor) List(ctx context.Context, path string, options cloud.ListOptions) ([]*cloud.FileInfo, error) {
	// Resolve path
	prefix := a.ResolvePath(path)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var files []*cloud.FileInfo
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            aws.String(a.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		}

		if options.MaxKeys > 0 {
			// Ensure MaxKeys doesn't overflow int32
			maxKeys := options.MaxKeys
			if maxKeys > 2147483647 {
				maxKeys = 2147483647
			}
			input.MaxKeys = aws.Int32(int32(maxKeys)) // #nosec G115 - bounds check performed above
		}

		if !options.Recursive {
			input.Delimiter = aws.String("/")
		}

		if options.StartAfter != "" {
			input.StartAfter = aws.String(options.StartAfter)
		}

		resp, err := a.client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		// Add files
		for _, obj := range resp.Contents {
			files = append(files, &cloud.FileInfo{
				Path:         a.UnresolvePath(*obj.Key),
				Size:         *obj.Size,
				ModTime:      *obj.LastModified,
				IsDir:        false,
				ETag:         strings.Trim(*obj.ETag, "\""),
				StorageClass: string(obj.StorageClass),
			})
		}

		// Add directories (common prefixes)
		for _, prefix := range resp.CommonPrefixes {
			dirPath := strings.TrimSuffix(*prefix.Prefix, "/")
			files = append(files, &cloud.FileInfo{
				Path:    a.UnresolvePath(dirPath),
				IsDir:   true,
				ModTime: time.Now(),
			})
		}

		if !aws.ToBool(resp.IsTruncated) {
			break
		}

		continuationToken = resp.NextContinuationToken
	}

	return files, nil
}

// Read downloads a file from S3
func (a *Adaptor) Read(ctx context.Context, path string, options cloud.DownloadOptions) (io.ReadCloser, error) {
	key := a.ResolvePath(path)

	input := &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}

	if options.Range != "" {
		input.Range = aws.String(options.Range)
	}

	if options.IfMatch != "" {
		input.IfMatch = aws.String(options.IfMatch)
	}

	if !options.IfModified.IsZero() {
		input.IfModifiedSince = aws.Time(options.IfModified)
	}

	resp, err := a.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	return resp.Body, nil
}

// Write uploads a file to S3
func (a *Adaptor) Write(ctx context.Context, path string, reader io.Reader, options cloud.UploadOptions) error {
	key := a.ResolvePath(path)

	input := &s3.PutObjectInput{
		Bucket:   aws.String(a.bucket),
		Key:      aws.String(key),
		Body:     reader,
		Metadata: options.Metadata,
	}

	if options.ContentType != "" {
		input.ContentType = aws.String(options.ContentType)
	}

	if options.StorageClass != "" {
		input.StorageClass = types.StorageClass(options.StorageClass)
	}

	if options.CacheControl != "" {
		input.CacheControl = aws.String(options.CacheControl)
	}

	_, err := a.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

// Delete removes a file from S3
func (a *Adaptor) Delete(ctx context.Context, path string) error {
	key := a.ResolvePath(path)

	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}

// Move moves a file within S3
func (a *Adaptor) Move(ctx context.Context, oldPath, newPath string) error {
	// Copy to new location
	if err := a.Copy(ctx, oldPath, newPath); err != nil {
		return err
	}

	// Delete from old location
	return a.Delete(ctx, oldPath)
}

// Copy copies a file within S3
func (a *Adaptor) Copy(ctx context.Context, srcPath, dstPath string) error {
	srcKey := a.ResolvePath(srcPath)
	dstKey := a.ResolvePath(dstPath)

	copySource := fmt.Sprintf("%s/%s", a.bucket, srcKey)

	_, err := a.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(a.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(dstKey),
	})

	if err != nil {
		return fmt.Errorf("copy object: %w", err)
	}

	return nil
}

// CreateDirectory creates a directory (S3 doesn't have real directories)
func (a *Adaptor) CreateDirectory(ctx context.Context, path string) error {
	// In S3, directories are just prefixes, so we create an empty object with trailing slash
	key := a.ResolvePath(path)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(""),
	})

	if err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return nil
}

// GetSignedURL generates a pre-signed URL for temporary access
func (a *Adaptor) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	key := a.ResolvePath(path)

	presignClient := s3.NewPresignClient(a.client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", fmt.Errorf("presign URL: %w", err)
	}

	return req.URL, nil
}

// buildAWSConfig builds AWS configuration from AdaptorConfig
func (a *Adaptor) buildAWSConfig(ctx context.Context, cfg cloud.AdaptorConfig) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error

	// Set region
	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	// Set endpoint for S3-compatible services
	if cfg.Endpoint != "" {
		// Use BaseEndpoint for custom endpoints in SDK v2
		opts = append(opts, config.WithBaseEndpoint(cfg.Endpoint))
	}

	// Set credentials if provided
	if cfg.Credentials != nil {
		if accessKey, ok := cfg.Credentials["access_key"].(string); ok {
			if secretKey, ok := cfg.Credentials["secret_key"].(string); ok {
				opts = append(opts, config.WithCredentialsProvider(
					credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
				))
			}
		}
	}

	return config.LoadDefaultConfig(ctx, opts...)
}
