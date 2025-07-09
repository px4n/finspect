package cloud

import (
	"fmt"
	"path"
	"strings"
)

// BaseCloudAdaptor provides common functionality for cloud adaptors
type BaseCloudAdaptor struct {
	provider Provider
	bucket   string
	prefix   string
}

// NewBaseCloudAdaptor creates a new base cloud adaptor
func NewBaseCloudAdaptor(provider Provider) *BaseCloudAdaptor {
	return &BaseCloudAdaptor{
		provider: provider,
	}
}

// GetProvider returns the cloud provider type
func (b *BaseCloudAdaptor) GetProvider() Provider {
	return b.provider
}

// SetBucket sets the bucket name
func (b *BaseCloudAdaptor) SetBucket(bucket string) {
	b.bucket = bucket
}

// SetPrefix sets the path prefix
func (b *BaseCloudAdaptor) SetPrefix(prefix string) {
	b.prefix = strings.TrimSuffix(prefix, "/")
}

// ResolvePath resolves a path relative to the adaptor's prefix
func (b *BaseCloudAdaptor) ResolvePath(p string) string {
	// Remove leading slash
	p = strings.TrimPrefix(p, "/")

	if b.prefix == "" {
		return p
	}

	return path.Join(b.prefix, p)
}

// UnresolvePath removes the prefix from a path
func (b *BaseCloudAdaptor) UnresolvePath(p string) string {
	if b.prefix == "" {
		return "/" + p
	}

	p = strings.TrimPrefix(p, b.prefix)
	p = strings.TrimPrefix(p, "/")

	if p == "" {
		return "/"
	}

	return "/" + p
}

// ValidatePath validates a cloud storage path
func (b *BaseCloudAdaptor) ValidatePath(p string) error {
	if p == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for invalid characters
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path cannot contain backslashes")
	}

	// Check for double slashes
	if strings.Contains(p, "//") {
		return fmt.Errorf("path cannot contain double slashes")
	}

	return nil
}

// IsDirectory determines if a path represents a directory
func (b *BaseCloudAdaptor) IsDirectory(p string) bool {
	return strings.HasSuffix(p, "/")
}

// NormalizePath normalizes a cloud storage path
func (b *BaseCloudAdaptor) NormalizePath(p string) string {
	// Trim leading and trailing slashes
	p = strings.Trim(p, "/")

	// Replace multiple slashes with single slash
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}

	return p
}

// GetBucketAndKey extracts bucket and key from a path
func (b *BaseCloudAdaptor) GetBucketAndKey(p string) (bucket, key string) {
	if b.bucket != "" {
		return b.bucket, b.ResolvePath(p)
	}

	// If no bucket is set, try to extract from path
	parts := strings.SplitN(p, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return parts[0], ""
}

// NOTE: The DefaultCloudAdaptorFactory implementation has been moved to factory.go
// to avoid import cycles
