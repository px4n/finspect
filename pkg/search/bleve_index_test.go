package search_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/px4n/finspect/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBleveIndex(t *testing.T) {
	// Create temporary index
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	index, err := search.NewBleveIndex(indexPath)
	require.NoError(t, err)
	defer index.Close()

	ctx := context.Background()

	// Create test metadata
	meta1 := &search.IndexableMetadata{
		Adaptor:     "filesystem",
		Path:        "/docs/readme.md",
		Description: "Project documentation",
		Tags:        []string{"docs", "markdown"},
		ContentType: "text/markdown",
		Size:        1024,
		ModTime:     time.Now(),
		IsDir:       false,
	}

	meta2 := &search.IndexableMetadata{
		Adaptor:     "filesystem",
		Path:        "/src/main.go",
		Description: "Main application file",
		Tags:        []string{"source", "golang"},
		ContentType: "text/x-go",
		Size:        2048,
		ModTime:     time.Now(),
		IsDir:       false,
	}

	// Index documents
	err = index.Index(ctx, meta1)
	assert.NoError(t, err)

	err = index.Index(ctx, meta2)
	assert.NoError(t, err)

	// Verify documents were indexed
	stats, err := index.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), stats.DocumentCount)

	// Give Bleve a moment to finish indexing
	time.Sleep(50 * time.Millisecond)

	// Search for documents
	result, err := index.Search(ctx, "documentation")
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), result.Total)
	if assert.Len(t, result.Hits, 1) {
		assert.Equal(t, "/docs/readme.md", result.Hits[0].Document.Path)
	}

	// Search by description text
	result, err = index.Search(ctx, "Main application")
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), result.Total)
	if assert.Len(t, result.Hits, 1) {
		assert.Equal(t, "/src/main.go", result.Hits[0].Document.Path)
	}

	// Search all
	result, err = index.Search(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), result.Total)
}

func TestBleveIndexWithContent(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	index, err := search.NewBleveIndex(indexPath)
	require.NoError(t, err)
	defer index.Close()

	ctx := context.Background()

	meta := &search.IndexableMetadata{
		Adaptor:     "filesystem",
		Path:        "/notes/todo.txt",
		ContentType: "text/plain",
		Size:        100,
		ModTime:     time.Now(),
	}

	content := "Remember to implement the search feature with Bleve"

	err = index.IndexWithContent(ctx, meta, content)
	assert.NoError(t, err)

	// Search content
	result, err := index.Search(ctx, "Bleve")
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), result.Total)
	assert.Equal(t, "/notes/todo.txt", result.Hits[0].Document.Path)
}

func TestBleveDelete(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	index, err := search.NewBleveIndex(indexPath)
	require.NoError(t, err)
	defer index.Close()

	ctx := context.Background()

	// Index a document
	meta := &search.IndexableMetadata{
		Adaptor:     "filesystem",
		Path:        "/test.txt",
		Description: "Test document",
		ContentType: "text/plain",
		ModTime:     time.Now(),
	}
	err = index.Index(ctx, meta)
	require.NoError(t, err)

	// Delete a document
	err = index.Delete(ctx, "filesystem", "/test.txt")
	assert.NoError(t, err)

	// Verify it's gone
	result, err := index.Search(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), result.Total)
}

func TestBleveBatchIndex(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	index, err := search.NewBleveIndex(indexPath)
	require.NoError(t, err)
	defer index.Close()

	ctx := context.Background()

	metas := []*search.IndexableMetadata{
		{
			Adaptor:     "s3",
			Path:        "/bucket/file1.txt",
			ContentType: "text/plain",
			Size:        1000,
			ModTime:     time.Now(),
		},
		{
			Adaptor:     "s3",
			Path:        "/bucket/file2.txt",
			ContentType: "text/plain",
			Size:        2000,
			ModTime:     time.Now(),
		},
		{
			Adaptor:     "s3",
			Path:        "/bucket/file3.txt",
			ContentType: "text/plain",
			Size:        3000,
			ModTime:     time.Now(),
		},
	}

	err = index.BatchIndex(ctx, metas)
	assert.NoError(t, err)

	// Search by adaptor
	result, err := index.Search(ctx, "", search.WithAdaptor("s3"))
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), result.Total)
}

func TestBleveSearchWithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	index, err := search.NewBleveIndex(indexPath)
	require.NoError(t, err)
	defer index.Close()

	ctx := context.Background()

	// Index test documents
	testMetas := []*search.IndexableMetadata{
		{
			Adaptor:     "s3",
			Path:        "/bucket/file1.txt",
			ContentType: "text/plain",
			Size:        1000,
			ModTime:     time.Now(),
		},
		{
			Adaptor:     "s3",
			Path:        "/bucket/file2.txt",
			ContentType: "text/plain",
			Size:        2000,
			ModTime:     time.Now(),
		},
		{
			Adaptor:     "filesystem",
			Path:        "/other/file.jpg",
			ContentType: "image/jpeg",
			Size:        3000,
			ModTime:     time.Now(),
		},
	}

	err = index.BatchIndex(ctx, testMetas)
	assert.NoError(t, err)

	// Give Bleve a moment to finish indexing
	time.Sleep(50 * time.Millisecond)

	// Verify documents are indexed
	stats, err := index.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), stats.DocumentCount)

	// Search with pagination
	result, err := index.Search(ctx, "",
		search.WithLimit(2),
		search.WithOffset(1),
	)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(result.Hits), 2)

	// Test individual filters first
	result, err = index.Search(ctx, "", search.WithContentType("text/plain"))
	assert.NoError(t, err)
	t.Logf("ContentType filter only: %d results", result.Total)

	result, err = index.Search(ctx, "", search.WithPathPrefix("/bucket/"))
	assert.NoError(t, err)
	t.Logf("PathPrefix filter only: %d results", result.Total)

	// Search with both filters
	result, err = index.Search(ctx, "",
		search.WithContentType("text/plain"),
		search.WithPathPrefix("/bucket/"),
	)
	assert.NoError(t, err)
	t.Logf("Both filters: %d results", result.Total)
	assert.Greater(t, result.Total, uint64(0))

	// Search with facets
	result, err = index.Search(ctx, "",
		search.WithFacets(true),
	)
	assert.NoError(t, err)
	assert.NotNil(t, result.Facets)
	assert.Contains(t, result.Facets, "content_type")
	assert.Contains(t, result.Facets, "adaptor")
}

func TestBleveGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	index, err := search.NewBleveIndex(indexPath)
	require.NoError(t, err)
	defer index.Close()

	stats, err := index.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), stats.DocumentCount)
	assert.Equal(t, indexPath, stats.IndexPath)
	assert.GreaterOrEqual(t, stats.IndexSize, int64(0))
}

func TestBleveIndexHighlighting(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "test.bleve")

	index, err := search.NewBleveIndex(indexPath)
	require.NoError(t, err)
	defer index.Close()

	ctx := context.Background()

	// Index document with content
	meta := &search.IndexableMetadata{
		Adaptor:     "filesystem",
		Path:        "/example.txt",
		Description: "This is an example document for testing search highlighting",
		ContentType: "text/plain",
		ModTime:     time.Now(),
	}

	content := "The quick brown fox jumps over the lazy dog. This sentence contains the word example."

	err = index.IndexWithContent(ctx, meta, content)
	require.NoError(t, err)

	// Search with highlighting
	result, err := index.Search(ctx, "example", search.WithHighlight(true))
	require.NoError(t, err)
	require.Equal(t, uint64(1), result.Total)
	require.Len(t, result.Hits, 1)

	hit := result.Hits[0]
	assert.NotNil(t, hit.Highlights)
	assert.Contains(t, hit.Highlights, "description")
	assert.Contains(t, hit.Highlights, "content")
}
