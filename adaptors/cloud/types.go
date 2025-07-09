package cloud

import (
	"context"
	"io"
	"time"
)

// Provider represents different cloud storage providers
type Provider string

const (
	// ProviderS3 represents Amazon S3 or S3-compatible storage
	ProviderS3 Provider = "s3"
	// ProviderGCS represents Google Cloud Storage
	ProviderGCS Provider = "gcs"
	// ProviderAzure represents Azure Blob Storage
	ProviderAzure Provider = "azure"
	// ProviderGoogleDrive represents Google Drive storage
	ProviderGoogleDrive Provider = "googledrive"
)

// AdaptorConfig is the common configuration for cloud adaptors
type AdaptorConfig struct {
	Provider    Provider               `json:"provider"`
	Bucket      string                 `json:"bucket,omitempty"`      // For S3, GCS, Azure
	Region      string                 `json:"region,omitempty"`      // For S3
	Endpoint    string                 `json:"endpoint,omitempty"`    // For S3-compatible services
	Credentials map[string]interface{} `json:"credentials,omitempty"` // Provider-specific credentials
}

// FileInfo represents file information from cloud storage
type FileInfo struct {
	Path         string
	Size         int64
	ModTime      time.Time
	IsDir        bool
	ETag         string
	ContentType  string
	StorageClass string
	Metadata     map[string]string
}

// ListOptions provides options for listing cloud objects
type ListOptions struct {
	Prefix     string
	Delimiter  string
	MaxKeys    int
	StartAfter string
	Recursive  bool
}

// UploadOptions provides options for uploading objects
type UploadOptions struct {
	ContentType  string
	StorageClass string
	Metadata     map[string]string
	CacheControl string
}

// DownloadOptions provides options for downloading objects
type DownloadOptions struct {
	Range      string // HTTP Range header format
	IfMatch    string // ETag for conditional download
	IfModified time.Time
}

// Adaptor is the interface that all cloud storage adaptors must implement
type Adaptor interface {
	// Connect establishes connection to the cloud provider
	Connect(ctx context.Context, config AdaptorConfig) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// Stat returns information about a file or directory
	Stat(ctx context.Context, path string) (*FileInfo, error)

	// List returns a list of files in a directory
	List(ctx context.Context, path string, options ListOptions) ([]*FileInfo, error)

	// Read downloads a file from cloud storage
	Read(ctx context.Context, path string, options DownloadOptions) (io.ReadCloser, error)

	// Write uploads a file to cloud storage
	Write(ctx context.Context, path string, reader io.Reader, options UploadOptions) error

	// Delete removes a file from cloud storage
	Delete(ctx context.Context, path string) error

	// Move moves a file within cloud storage
	Move(ctx context.Context, oldPath, newPath string) error

	// Copy copies a file within cloud storage
	Copy(ctx context.Context, srcPath, dstPath string) error

	// CreateDirectory creates a directory (if supported by the provider)
	CreateDirectory(ctx context.Context, path string) error

	// GetSignedURL generates a pre-signed URL for temporary access
	GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// GetProvider returns the cloud provider type
	GetProvider() Provider
}

// AdaptorFactory creates cloud adaptors based on provider type
type AdaptorFactory interface {
	Create(provider Provider) (Adaptor, error)
}
