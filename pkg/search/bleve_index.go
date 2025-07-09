package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
)

// BleveIndex implements full-text search using Bleve.
type BleveIndex struct {
	index bleve.Index
	path  string
	mu    sync.RWMutex
}

// Document represents a searchable document in the index.
type Document struct {
	ID          string    `json:"id"`
	Adaptor     string    `json:"adaptor"`
	Path        string    `json:"path"`
	Content     string    `json:"content,omitempty"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`
}

// NewBleveIndex creates a new Bleve index.
func NewBleveIndex(indexPath string) (*BleveIndex, error) {
	// Check if index already exists
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index with custom mapping
		mapping := buildIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("create index: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}

	return &BleveIndex{
		index: index,
		path:  indexPath,
	}, nil
}

// buildIndexMapping creates the index mapping for documents.
func buildIndexMapping() mapping.IndexMapping {
	// Create a custom document mapping
	docMapping := bleve.NewDocumentMapping()

	// Text fields with stemming and full-text search
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = "standard"
	textFieldMapping.Store = true
	textFieldMapping.Index = true
	textFieldMapping.IncludeTermVectors = true
	textFieldMapping.IncludeInAll = true

	// Keyword fields for exact matching
	keywordFieldMapping := bleve.NewKeywordFieldMapping()
	keywordFieldMapping.Store = true
	keywordFieldMapping.Index = true
	keywordFieldMapping.IncludeInAll = false

	// Numeric fields
	numericFieldMapping := bleve.NewNumericFieldMapping()
	numericFieldMapping.Store = true
	numericFieldMapping.Index = true
	numericFieldMapping.IncludeInAll = false

	// Date fields
	dateFieldMapping := bleve.NewDateTimeFieldMapping()
	dateFieldMapping.Store = true
	dateFieldMapping.Index = true
	dateFieldMapping.IncludeInAll = false

	// Boolean fields
	boolFieldMapping := bleve.NewBooleanFieldMapping()
	boolFieldMapping.Store = true
	boolFieldMapping.Index = true
	boolFieldMapping.IncludeInAll = false

	// Special mapping for path - use keyword for exact/prefix matching
	pathFieldMapping := bleve.NewKeywordFieldMapping()
	pathFieldMapping.Store = true
	pathFieldMapping.Index = true
	pathFieldMapping.IncludeInAll = true

	// Add field mappings
	docMapping.AddFieldMappingsAt("id", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("adaptor", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("path", pathFieldMapping)
	docMapping.AddFieldMappingsAt("content", textFieldMapping)
	docMapping.AddFieldMappingsAt("description", textFieldMapping)
	docMapping.AddFieldMappingsAt("tags", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("content_type", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("size", numericFieldMapping)
	docMapping.AddFieldMappingsAt("mod_time", dateFieldMapping)
	docMapping.AddFieldMappingsAt("is_dir", boolFieldMapping)

	// Create index mapping
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = docMapping
	indexMapping.DefaultAnalyzer = "standard"

	return indexMapping
}

// Close closes the index.
func (b *BleveIndex) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.index.Close()
}

// Index adds or updates a document in the index.
func (b *BleveIndex) Index(_ context.Context, meta *IndexableMetadata) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	doc := &Document{
		ID:          fmt.Sprintf("%s:%s", meta.Adaptor, meta.Path),
		Adaptor:     meta.Adaptor,
		Path:        meta.Path,
		Description: meta.Description,
		Tags:        meta.Tags,
		ContentType: meta.ContentType,
		Size:        meta.Size,
		ModTime:     meta.ModTime,
		IsDir:       meta.IsDir,
	}

	return b.index.Index(doc.ID, doc)
}

// IndexWithContent adds or updates a document with content in the index.
func (b *BleveIndex) IndexWithContent(_ context.Context, meta *IndexableMetadata, content string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	doc := &Document{
		ID:          fmt.Sprintf("%s:%s", meta.Adaptor, meta.Path),
		Adaptor:     meta.Adaptor,
		Path:        meta.Path,
		Content:     content,
		Description: meta.Description,
		Tags:        meta.Tags,
		ContentType: meta.ContentType,
		Size:        meta.Size,
		ModTime:     meta.ModTime,
		IsDir:       meta.IsDir,
	}

	return b.index.Index(doc.ID, doc)
}

// Delete removes a document from the index.
func (b *BleveIndex) Delete(_ context.Context, adaptor, path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := fmt.Sprintf("%s:%s", adaptor, path)
	return b.index.Delete(id)
}

// Search performs a search query.
func (b *BleveIndex) Search(ctx context.Context, searchQuery string, options ...Option) (*Result, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Apply options
	opts := defaultSearchOptions()
	for _, opt := range options {
		opt(opts)
	}

	// Build and execute search
	searchResult, err := b.executeSearch(ctx, searchQuery, opts)
	if err != nil {
		return nil, err
	}

	// Convert results
	return b.convertResults(searchResult), nil
}

// buildQuery builds the search query from options
func (b *BleveIndex) buildQuery(searchQuery string, opts *Options) query.Query {
	var queries []query.Query

	// Add text search query
	if searchQuery != "" {
		queries = append(queries, b.buildTextQuery(searchQuery))
	}

	// Add filter queries
	b.addFilterQueries(&queries, opts)

	// Combine all queries
	switch len(queries) {
	case 0:
		return bleve.NewMatchAllQuery()
	case 1:
		return queries[0]
	default:
		return bleve.NewConjunctionQuery(queries...)
	}
}

// buildTextQuery builds multi-field text search query
func (b *BleveIndex) buildTextQuery(searchQuery string) query.Query {
	pathMatch := bleve.NewMatchQuery(searchQuery)
	pathMatch.SetField("path")

	contentMatch := bleve.NewMatchQuery(searchQuery)
	contentMatch.SetField("content")

	descMatch := bleve.NewMatchQuery(searchQuery)
	descMatch.SetField("description")

	pathPhrase := bleve.NewMatchPhraseQuery(searchQuery)
	pathPhrase.SetField("path")

	contentPhrase := bleve.NewMatchPhraseQuery(searchQuery)
	contentPhrase.SetField("content")

	return bleve.NewDisjunctionQuery(
		pathMatch,
		contentMatch,
		descMatch,
		pathPhrase,
		contentPhrase,
	)
}

// addFilterQueries adds filter queries based on options
func (b *BleveIndex) addFilterQueries(queries *[]query.Query, opts *Options) {
	if opts.Adaptor != "" {
		adaptorQuery := bleve.NewTermQuery(opts.Adaptor)
		adaptorQuery.SetField("adaptor")
		*queries = append(*queries, adaptorQuery)
	}

	if opts.ContentType != "" {
		ctQuery := bleve.NewTermQuery(opts.ContentType)
		ctQuery.SetField("content_type")
		*queries = append(*queries, ctQuery)
	}

	if len(opts.Tags) > 0 {
		for _, tag := range opts.Tags {
			tagQuery := bleve.NewTermQuery(tag)
			tagQuery.SetField("tags")
			*queries = append(*queries, tagQuery)
		}
	}

	if opts.PathPrefix != "" {
		prefixQuery := bleve.NewPrefixQuery(opts.PathPrefix)
		prefixQuery.SetField("path")
		*queries = append(*queries, prefixQuery)
	}
}

// BatchIndex indexes multiple documents.
func (b *BleveIndex) BatchIndex(_ context.Context, metas []*IndexableMetadata) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	batch := b.index.NewBatch()
	for _, meta := range metas {
		doc := &Document{
			ID:          fmt.Sprintf("%s:%s", meta.Adaptor, meta.Path),
			Adaptor:     meta.Adaptor,
			Path:        meta.Path,
			Description: meta.Description,
			Tags:        meta.Tags,
			ContentType: meta.ContentType,
			Size:        meta.Size,
			ModTime:     meta.ModTime,
			IsDir:       meta.IsDir,
		}
		if err := batch.Index(doc.ID, doc); err != nil {
			return fmt.Errorf("batch index document: %w", err)
		}
	}

	return b.index.Batch(batch)
}

// GetStats returns index statistics.
func (b *BleveIndex) GetStats() (*IndexStats, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	docCount, err := b.index.DocCount()
	if err != nil {
		return nil, err
	}

	indexPath := b.path
	indexSize := getDirectorySize(indexPath)

	return &IndexStats{
		DocumentCount: docCount,
		IndexSize:     indexSize,
		IndexPath:     indexPath,
	}, nil
}

// getDirectorySize calculates the total size of a directory.
func getDirectorySize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// executeSearch builds and executes the search request
func (b *BleveIndex) executeSearch(
	ctx context.Context,
	searchQuery string,
	opts *Options,
) (*bleve.SearchResult, error) {
	// Build the query
	finalQuery := b.buildQuery(searchQuery, opts)

	// Create search request
	searchRequest := bleve.NewSearchRequestOptions(finalQuery, opts.Limit, opts.Offset, false)
	searchRequest.Fields = []string{"*"}

	// Configure search options
	b.configureSearchRequest(searchRequest, opts)

	// Execute search
	searchResult, err := b.index.SearchInContext(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return searchResult, nil
}

// configureSearchRequest configures the search request with options
func (b *BleveIndex) configureSearchRequest(searchRequest *bleve.SearchRequest, opts *Options) {
	// Add facets if requested
	if opts.IncludeFacets {
		searchRequest.AddFacet("content_type", bleve.NewFacetRequest("content_type", 10))
		searchRequest.AddFacet("adaptor", bleve.NewFacetRequest("adaptor", 10))
	}

	// Add sorting
	switch opts.SortBy {
	case "size":
		searchRequest.SortBy([]string{"-size"})
	case "modified":
		searchRequest.SortBy([]string{"-mod_time"})
	case "path":
		searchRequest.SortBy([]string{"path"})
	}

	// Add highlighting
	if opts.Highlight {
		searchRequest.Highlight = bleve.NewHighlightWithStyle("html")
		searchRequest.Highlight.AddField("content")
		searchRequest.Highlight.AddField("path")
		searchRequest.Highlight.AddField("description")
	}
}

// convertResults converts Bleve search results to our Result type
func (b *BleveIndex) convertResults(searchResult *bleve.SearchResult) *Result {
	result := &Result{
		Total:    searchResult.Total,
		Duration: searchResult.Took,
		Hits:     make([]*Hit, 0, len(searchResult.Hits)),
	}

	// Convert hits
	for _, hit := range searchResult.Hits {
		h := b.convertHit(hit)
		result.Hits = append(result.Hits, h)
	}

	// Convert facets
	if searchResult.Facets != nil {
		result.Facets = b.convertFacets(searchResult.Facets)
	}

	return result
}

// convertHit converts a single Bleve hit to our Hit type
func (b *BleveIndex) convertHit(hit *search.DocumentMatch) *Hit {
	h := &Hit{
		ID:    hit.ID,
		Score: hit.Score,
		Document: &Document{
			ID: hit.ID,
		},
	}

	// Extract document fields
	b.extractDocumentFields(h.Document, hit.Fields)

	// Add highlights
	if hit.Fragments != nil {
		h.Highlights = make(map[string][]string)
		for field, fragments := range hit.Fragments {
			h.Highlights[field] = fragments
		}
	}

	return h
}

// extractDocumentFields extracts fields from Bleve result into Document
func (b *BleveIndex) extractDocumentFields(doc *Document, fields map[string]interface{}) {
	if adaptor, ok := fields["adaptor"].(string); ok {
		doc.Adaptor = adaptor
	}
	if path, ok := fields["path"].(string); ok {
		doc.Path = path
	}
	if contentType, ok := fields["content_type"].(string); ok {
		doc.ContentType = contentType
	}
	if size, ok := fields["size"].(float64); ok {
		doc.Size = int64(size)
	}
	if description, ok := fields["description"].(string); ok {
		doc.Description = description
	}
	if tags, ok := fields["tags"].([]interface{}); ok {
		doc.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if s, ok := tag.(string); ok {
				doc.Tags[i] = s
			}
		}
	}
	if isDir, ok := fields["is_dir"].(bool); ok {
		doc.IsDir = isDir
	}
	if modTime, ok := fields["mod_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, modTime); err == nil {
			doc.ModTime = t
		}
	}
}

// convertFacets converts Bleve facets to our Facet type
func (b *BleveIndex) convertFacets(facets search.FacetResults) map[string]*Facet {
	result := make(map[string]*Facet)
	for name, facetResult := range facets {
		facet := &Facet{
			Field: name,
			Terms: make([]*FacetTerm, 0),
		}
		for _, term := range facetResult.Terms.Terms() {
			facet.Terms = append(facet.Terms, &FacetTerm{
				Term:  term.Term,
				Count: term.Count,
			})
		}
		result[name] = facet
	}
	return result
}
