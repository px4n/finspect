package search

import (
	"context"
	"time"
)

// Result represents search results.
type Result struct {
	Total    uint64            `json:"total"`
	Duration time.Duration     `json:"duration"`
	Hits     []*Hit            `json:"hits"`
	Facets   map[string]*Facet `json:"facets,omitempty"`
}

// Hit represents a single search hit.
type Hit struct {
	ID         string              `json:"id"`
	Score      float64             `json:"score"`
	Document   *Document           `json:"document"`
	Highlights map[string][]string `json:"highlights,omitempty"`
}

// Facet represents faceted search results.
type Facet struct {
	Field string       `json:"field"`
	Terms []*FacetTerm `json:"terms"`
}

// FacetTerm represents a term in a facet.
type FacetTerm struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}

// Options contains search parameters.
type Options struct {
	// Query options
	Limit  int
	Offset int

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

	// Features
	Highlight     bool
	IncludeFacets bool

	// Sorting
	SortBy    string // "relevance", "path", "size", "modified"
	SortOrder string // "asc", "desc"
}

// Option is a function that modifies search options.
type Option func(*Options)

// defaultSearchOptions returns default search options.
func defaultSearchOptions() *Options {
	return &Options{
		Limit:     20,
		Offset:    0,
		SortBy:    "relevance",
		SortOrder: "desc",
		Highlight: true,
	}
}

// WithLimit sets the result limit.
func WithLimit(limit int) Option {
	return func(o *Options) {
		o.Limit = limit
	}
}

// WithOffset sets the result offset.
func WithOffset(offset int) Option {
	return func(o *Options) {
		o.Offset = offset
	}
}

// WithAdaptor filters by adaptor.
func WithAdaptor(adaptor string) Option {
	return func(o *Options) {
		o.Adaptor = adaptor
	}
}

// WithPathPrefix filters by path prefix.
func WithPathPrefix(prefix string) Option {
	return func(o *Options) {
		o.PathPrefix = prefix
	}
}

// WithContentType filters by content type.
func WithContentType(contentType string) Option {
	return func(o *Options) {
		o.ContentType = contentType
	}
}

// WithTags filters by tags.
func WithTags(tags ...string) Option {
	return func(o *Options) {
		o.Tags = tags
	}
}

// WithSizeRange filters by size range.
func WithSizeRange(minSize, maxSize int64) Option {
	return func(o *Options) {
		o.MinSize = minSize
		o.MaxSize = maxSize
	}
}

// WithModifiedRange filters by modification time range.
func WithModifiedRange(after, before *time.Time) Option {
	return func(o *Options) {
		o.ModifiedAfter = after
		o.ModifiedBefore = before
	}
}

// WithHighlight enables search result highlighting.
func WithHighlight(enable bool) Option {
	return func(o *Options) {
		o.Highlight = enable
	}
}

// WithFacets enables faceted search.
func WithFacets(enable bool) Option {
	return func(o *Options) {
		o.IncludeFacets = enable
	}
}

// WithSort sets the sort order.
func WithSort(by, order string) Option {
	return func(o *Options) {
		o.SortBy = by
		o.SortOrder = order
	}
}

// IndexStats represents index statistics.
type IndexStats struct {
	DocumentCount uint64 `json:"document_count"`
	IndexSize     int64  `json:"index_size"`
	IndexPath     string `json:"index_path"`
}

// IndexableMetadata represents metadata that can be indexed.
type IndexableMetadata struct {
	Adaptor     string
	Path        string
	Description string
	Tags        []string
	ContentType string
	Size        int64
	ModTime     time.Time
	IsDir       bool
}

// Indexer defines the interface for search indexing.
type Indexer interface {
	// Index adds or updates a document
	Index(ctx context.Context, meta *IndexableMetadata) error

	// IndexWithContent adds or updates a document with content
	IndexWithContent(ctx context.Context, meta *IndexableMetadata, content string) error

	// Delete removes a document
	Delete(ctx context.Context, adaptor, path string) error

	// BatchIndex indexes multiple documents
	BatchIndex(ctx context.Context, metas []*IndexableMetadata) error

	// Search performs a search
	Search(ctx context.Context, query string, options ...Option) (*Result, error)

	// GetStats returns index statistics
	GetStats() (*IndexStats, error)

	// Close closes the index
	Close() error
}
