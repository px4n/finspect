// Package metadata provides extended metadata storage and retrieval for finspect.
package metadata

import (
	"context"
	"time"
)

// Metadata represents extended metadata for a file or directory.
type Metadata struct {
	// Core identifiers
	Path      string    `json:"path"`       // VFS path
	Adaptor   string    `json:"adaptor"`    // Adaptor name (e.g., "filesystem", "s3")
	UpdatedAt time.Time `json:"updated_at"` // Last metadata update

	// File attributes
	Size        int64     `json:"size"`
	Mode        uint32    `json:"mode"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`
	ContentType string    `json:"content_type"` // MIME type

	// Content hashes
	MD5    string `json:"md5,omitempty"`    // MD5 hash
	SHA256 string `json:"sha256,omitempty"` // SHA256 hash

	// Extended attributes
	Tags        []string          `json:"tags,omitempty"`        // User-defined tags
	Description string            `json:"description,omitempty"` // User description
	Properties  map[string]string `json:"properties,omitempty"`  // Custom key-value pairs

	// Media metadata (for images, videos, etc.)
	MediaInfo *MediaInfo `json:"media_info,omitempty"`

	// Document metadata (for text files, PDFs, etc.)
	DocumentInfo *DocumentInfo `json:"document_info,omitempty"`
}

// MediaInfo contains metadata specific to media files.
type MediaInfo struct {
	Width       int          `json:"width,omitempty"`
	Height      int          `json:"height,omitempty"`
	Duration    float64      `json:"duration,omitempty"` // In seconds
	FrameRate   float64      `json:"frame_rate,omitempty"`
	BitRate     int64        `json:"bit_rate,omitempty"`
	Codec       string       `json:"codec,omitempty"`
	Artist      string       `json:"artist,omitempty"`
	Album       string       `json:"album,omitempty"`
	Title       string       `json:"title,omitempty"`
	Year        int          `json:"year,omitempty"`
	Genre       string       `json:"genre,omitempty"`
	Location    *GeoLocation `json:"location,omitempty"`
	CameraMake  string       `json:"camera_make,omitempty"`
	CameraModel string       `json:"camera_model,omitempty"`
	DateTaken   *time.Time   `json:"date_taken,omitempty"`
}

// GeoLocation represents geographic coordinates.
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// DocumentInfo contains metadata specific to documents.
type DocumentInfo struct {
	WordCount   int        `json:"word_count,omitempty"`
	PageCount   int        `json:"page_count,omitempty"`
	Author      string     `json:"author,omitempty"`
	Title       string     `json:"title,omitempty"`
	Subject     string     `json:"subject,omitempty"`
	Keywords    []string   `json:"keywords,omitempty"`
	Language    string     `json:"language,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
	Modified    *time.Time `json:"modified,omitempty"`
	Application string     `json:"application,omitempty"` // Creating application
}

// Store defines the interface for metadata storage backends.
type Store interface {
	// Initialize the store
	Init(ctx context.Context) error

	// Close the store
	Close() error

	// Get retrieves metadata for a path
	Get(ctx context.Context, adaptor, path string) (*Metadata, error)

	// Put stores or updates metadata
	Put(ctx context.Context, metadata *Metadata) error

	// Delete removes metadata for a path
	Delete(ctx context.Context, adaptor, path string) error

	// Search queries metadata
	Search(ctx context.Context, query Query) ([]*Metadata, error)

	// Batch operations
	BatchPut(ctx context.Context, metadata []*Metadata) error
	BatchDelete(ctx context.Context, paths []PathKey) error
}

// PathKey uniquely identifies a file in the metadata store.
type PathKey struct {
	Adaptor string
	Path    string
}

// Query represents a metadata search query.
type Query struct {
	// Text search
	Text string

	// Filters
	Adaptor        string
	PathPrefix     string
	ContentType    string
	Tags           []string
	MinSize        int64
	MaxSize        int64
	ModifiedAfter  *time.Time
	ModifiedBefore *time.Time
	IsDir          *bool

	// Pagination
	Offset int
	Limit  int

	// Sorting
	SortBy    string // "path", "size", "modified", "name"
	SortOrder string // "asc", "desc"
}

// Extractor defines the interface for metadata extraction.
type Extractor interface {
	// Extract metadata from a file
	Extract(ctx context.Context, path string, content []byte) (*Metadata, error)

	// CanExtract checks if this extractor can handle the file type
	CanExtract(contentType string) bool
}

// ExtractorRegistry manages metadata extractors.
type ExtractorRegistry struct {
	extractors []Extractor
}

// NewExtractorRegistry creates a new extractor registry.
func NewExtractorRegistry() *ExtractorRegistry {
	return &ExtractorRegistry{
		extractors: make([]Extractor, 0),
	}
}

// Register adds an extractor to the registry.
func (r *ExtractorRegistry) Register(extractor Extractor) {
	r.extractors = append(r.extractors, extractor)
}

// Extract runs all applicable extractors on a file.
func (r *ExtractorRegistry) Extract(
	ctx context.Context,
	path string,
	contentType string,
	content []byte,
) (*Metadata, error) {
	metadata := &Metadata{
		Path:        path,
		ContentType: contentType,
		UpdatedAt:   time.Now(),
	}

	// Run all applicable extractors
	for _, extractor := range r.extractors {
		if extractor.CanExtract(contentType) {
			extracted, err := extractor.Extract(ctx, path, content)
			if err != nil {
				continue // Skip failed extractors
			}
			// Merge extracted metadata
			mergeMetadata(metadata, extracted)
		}
	}

	return metadata, nil
}

// mergeMetadata merges extracted metadata into the base metadata.
func mergeMetadata(base, extracted *Metadata) {
	if extracted.MediaInfo != nil {
		base.MediaInfo = extracted.MediaInfo
	}
	if extracted.DocumentInfo != nil {
		base.DocumentInfo = extracted.DocumentInfo
	}
	if len(extracted.Tags) > 0 {
		base.Tags = append(base.Tags, extracted.Tags...)
	}
	if extracted.Description != "" {
		base.Description = extracted.Description
	}
	if len(extracted.Properties) > 0 {
		if base.Properties == nil {
			base.Properties = make(map[string]string)
		}
		for k, v := range extracted.Properties {
			base.Properties[k] = v
		}
	}
}
