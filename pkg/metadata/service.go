package metadata

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sync"
	"time"

	"github.com/px4n/finspect/pkg/search"
	"github.com/px4n/finspect/pkg/vfs"
)

// Service provides metadata operations for VFS adaptors.
type Service struct {
	store    Store
	index    search.Indexer
	registry *ExtractorRegistry
	mu       sync.RWMutex
	cache    map[string]*cacheEntry
}

type cacheEntry struct {
	metadata  *Metadata
	timestamp time.Time
}

// NewService creates a new metadata service.
func NewService(store Store) *Service {
	return &Service{
		store:    store,
		registry: NewDefaultExtractorRegistry(),
		cache:    make(map[string]*cacheEntry),
	}
}

// NewServiceWithIndex creates a new metadata service with search index.
func NewServiceWithIndex(store Store, index search.Indexer) *Service {
	return &Service{
		store:    store,
		index:    index,
		registry: NewDefaultExtractorRegistry(),
		cache:    make(map[string]*cacheEntry),
	}
}

// SetExtractorRegistry sets a custom extractor registry.
func (s *Service) SetExtractorRegistry(registry *ExtractorRegistry) {
	s.registry = registry
}

// Extract extracts metadata from a file.
func (s *Service) Extract(ctx context.Context, vfs vfs.VFS, adaptor, filePath string) (*Metadata, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", adaptor, filePath)
	s.mu.RLock()
	if entry, ok := s.cache[cacheKey]; ok {
		if time.Since(entry.timestamp) < 5*time.Minute {
			s.mu.RUnlock()
			return entry.metadata, nil
		}
	}
	s.mu.RUnlock()

	// Get file info
	info, err := vfs.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Create base metadata
	metadata := &Metadata{
		Adaptor:     adaptor,
		Path:        filePath,
		UpdatedAt:   time.Now(),
		Size:        info.Size(),
		Mode:        uint32(info.Mode()),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		ContentType: detectContentType(filePath),
	}

	// Extract additional metadata if it's a file
	if !info.IsDir() && s.registry != nil {
		// Try to open and read the file for content-based extraction
		file, err := vfs.Open(filePath)
		if err == nil {
			defer file.Close()

			// Read a small chunk for content detection and extraction
			buf := make([]byte, 8192) // Read more for better extraction
			n, _ := file.Read(buf)
			if n > 0 {
				// Extract additional metadata
				extracted, err := s.registry.Extract(ctx, filePath, metadata.ContentType, buf[:n])
				if err != nil {
					// Log error but continue - partial metadata is better than none
					fmt.Printf("Warning: metadata extraction failed for %s: %v\n", filePath, err)
				} else if extracted != nil {
					// Merge extracted metadata
					mergeMetadata(metadata, extracted)
				}
			}
		}
	}

	// Cache the result
	s.mu.Lock()
	s.cache[cacheKey] = &cacheEntry{
		metadata:  metadata,
		timestamp: time.Now(),
	}
	s.mu.Unlock()

	return metadata, nil
}

// Store stores metadata in the backend.
func (s *Service) Store(ctx context.Context, metadata *Metadata) error {
	// Store in database
	if err := s.store.Put(ctx, metadata); err != nil {
		return err
	}

	// Index for search if available
	if s.index != nil {
		// Convert to IndexableMetadata
		indexable := &search.IndexableMetadata{
			Adaptor:     metadata.Adaptor,
			Path:        metadata.Path,
			Description: metadata.Description,
			Tags:        metadata.Tags,
			ContentType: metadata.ContentType,
			Size:        metadata.Size,
			ModTime:     metadata.ModTime,
			IsDir:       metadata.IsDir,
		}
		if err := s.index.Index(ctx, indexable); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Warning: failed to index metadata for %s: %v\n", metadata.Path, err)
		}
	}

	return nil
}

// StoreWithContent stores metadata and indexes content for full-text search.
func (s *Service) StoreWithContent(ctx context.Context, metadata *Metadata, content io.Reader) error {
	// Store metadata
	if err := s.store.Put(ctx, metadata); err != nil {
		return err
	}

	// Index with content if available
	if s.index != nil && content != nil {
		// Read content for indexing (limit to 1MB to avoid memory issues)
		buf := make([]byte, 1024*1024)
		n, _ := content.Read(buf)
		if n > 0 {
			// Convert to IndexableMetadata
			indexable := &search.IndexableMetadata{
				Adaptor:     metadata.Adaptor,
				Path:        metadata.Path,
				Description: metadata.Description,
				Tags:        metadata.Tags,
				ContentType: metadata.ContentType,
				Size:        metadata.Size,
				ModTime:     metadata.ModTime,
				IsDir:       metadata.IsDir,
			}
			if err := s.index.IndexWithContent(ctx, indexable, string(buf[:n])); err != nil {
				fmt.Printf("Warning: failed to index content for %s: %v\n", metadata.Path, err)
			}
		}
	}

	return nil
}

// Get retrieves metadata from the store.
func (s *Service) Get(ctx context.Context, adaptor, path string) (*Metadata, error) {
	return s.store.Get(ctx, adaptor, path)
}

// GetOrExtract retrieves metadata from store or extracts it if not found.
func (s *Service) GetOrExtract(ctx context.Context, vfs vfs.VFS, adaptor, filePath string) (*Metadata, error) {
	// Try to get from store first
	metadata, err := s.store.Get(ctx, adaptor, filePath)
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	// If found, check if it's still fresh
	if metadata != nil {
		statInfo, statErr := vfs.Stat(filePath)
		if statErr == nil && metadata.ModTime.Equal(statInfo.ModTime()) {
			return metadata, nil
		}
	}

	// Extract fresh metadata
	metadata, err = s.Extract(ctx, vfs, adaptor, filePath)
	if err != nil {
		return nil, fmt.Errorf("extract metadata: %w", err)
	}

	// Store it
	if err := s.store.Put(ctx, metadata); err != nil {
		// Log error but return the metadata anyway
		fmt.Printf("Warning: failed to store metadata for %s: %v\n", filePath, err)
	}

	return metadata, nil
}

// UpdateDirectory updates metadata for all files in a directory.
func (s *Service) UpdateDirectory(ctx context.Context, vfs vfs.VFS, adaptor, dirPath string) error {
	entries, err := vfs.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	batch := make([]*Metadata, 0, len(entries))
	for _, entry := range entries {
		filePath := path.Join(dirPath, entry.Name())

		// Extract metadata
		metadata, err := s.Extract(ctx, vfs, adaptor, filePath)
		if err != nil {
			fmt.Printf("Warning: failed to extract metadata for %s: %v\n", filePath, err)
			continue
		}

		batch = append(batch, metadata)

		// Recursively process subdirectories
		if entry.IsDir() {
			if err := s.UpdateDirectory(ctx, vfs, adaptor, filePath); err != nil {
				fmt.Printf("Warning: failed to process subdirectory %s: %v\n", filePath, err)
			}
		}
	}

	// Batch store metadata
	if len(batch) > 0 {
		if err := s.store.BatchPut(ctx, batch); err != nil {
			return fmt.Errorf("batch put metadata: %w", err)
		}
	}

	return nil
}

// Search searches for files matching the query.
func (s *Service) Search(ctx context.Context, query Query) ([]*Metadata, error) {
	return s.store.Search(ctx, query)
}

// FullTextSearch performs full-text search using the search index.
func (s *Service) FullTextSearch(ctx context.Context, query string, options ...search.Option) (*search.Result, error) {
	if s.index == nil {
		return nil, fmt.Errorf("search index not available")
	}
	return s.index.Search(ctx, query, options...)
}

// Delete removes metadata for a file.
func (s *Service) Delete(ctx context.Context, adaptor, path string) error {
	// Remove from cache
	cacheKey := fmt.Sprintf("%s:%s", adaptor, path)
	s.mu.Lock()
	delete(s.cache, cacheKey)
	s.mu.Unlock()

	// Remove from store
	if err := s.store.Delete(ctx, adaptor, path); err != nil {
		return err
	}

	// Remove from index if available
	if s.index != nil {
		if err := s.index.Delete(ctx, adaptor, path); err != nil {
			fmt.Printf("Warning: failed to delete from index %s:%s: %v\n", adaptor, path, err)
		}
	}

	return nil
}

// ClearCache clears the metadata cache.
func (s *Service) ClearCache() {
	s.mu.Lock()
	s.cache = make(map[string]*cacheEntry)
	s.mu.Unlock()
}

// WalkFunc is called for each file during a walk operation.
type WalkFunc func(path string, info fs.FileInfo, metadata *Metadata, err error) error

// Walk walks a directory tree and calls fn for each file with its metadata.
func (s *Service) Walk(ctx context.Context, vfs vfs.VFS, adaptor, root string, fn WalkFunc) error {
	return s.walk(ctx, vfs, adaptor, root, fn)
}

func (s *Service) walk(ctx context.Context, vfs vfs.VFS, adaptor, filePath string, fn WalkFunc) error {
	info, err := vfs.Stat(filePath)
	if err != nil {
		return fn(filePath, nil, nil, err)
	}

	// Get or extract metadata
	metadata, metaErr := s.GetOrExtract(ctx, vfs, adaptor, filePath)
	if metaErr != nil {
		// Call fn with the error but continue walking
		if fnErr := fn(filePath, info, nil, metaErr); fnErr != nil {
			return fnErr
		}
	} else {
		if fnErr := fn(filePath, info, metadata, nil); fnErr != nil {
			return fnErr
		}
	}

	// If it's a directory, walk its contents
	if info.IsDir() {
		entries, err := vfs.ReadDir(filePath)
		if err != nil {
			return fmt.Errorf("read directory %s: %w", filePath, err)
		}

		for _, entry := range entries {
			childPath := path.Join(filePath, entry.Name())
			if err := s.walk(ctx, vfs, adaptor, childPath, fn); err != nil {
				return err
			}
		}
	}

	return nil
}

// contentTypeMap maps file extensions to content types
var contentTypeMap = map[string]string{
	".txt":  "text/plain",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".xml":  "application/xml",
	".pdf":  "application/pdf",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".mp3":  "audio/mpeg",
	".mp4":  "video/mp4",
	".zip":  "application/zip",
	".tar":  "application/x-tar",
	".gz":   "application/gzip",
}

// detectContentType detects content type from file extension.
func detectContentType(filePath string) string {
	ext := path.Ext(filePath)
	if contentType, ok := contentTypeMap[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}
