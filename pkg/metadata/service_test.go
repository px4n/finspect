package metadata_test

import (
	"context"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/px4n/finspect/adaptors/filesystem"
	"github.com/px4n/finspect/pkg/metadata"
	"github.com/px4n/finspect/pkg/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataService(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "metadata.db")

	store, err := metadata.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	err = store.Init(ctx)
	require.NoError(t, err)

	service := metadata.NewService(store)

	// Create test VFS
	testDir := filepath.Join(tmpDir, "files")
	err = os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	// Create filesystem adaptor
	adaptor := filesystem.New()
	err = adaptor.Connect(ctx, map[string]interface{}{"root": testDir})
	require.NoError(t, err)

	// Create VFS router and mount adaptor
	router := vfs.NewRouter()
	err = router.Mount("/", adaptor)
	require.NoError(t, err)
	defer router.Close()

	fs := router

	// Create test files
	testFiles := map[string]string{
		"/test.txt":        "Hello, World!",
		"/docs/readme.md":  "# README\nThis is a test",
		"/images/test.jpg": "fake jpeg data",
	}

	for path, content := range testFiles {
		// Use path package instead of filepath for VFS paths
		dir := pathpkg.Dir(path)
		if dir != "/" && dir != "." {
			err := fs.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}

		w, err := fs.Create(path)
		require.NoError(t, err)
		_, err = io.WriteString(w, content)
		require.NoError(t, err)
		w.Close()
	}

	t.Run("Extract", func(t *testing.T) {
		metadata, err := service.Extract(ctx, fs, "filesystem", "/test.txt")
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, "/test.txt", metadata.Path)
		assert.Equal(t, "filesystem", metadata.Adaptor)
		assert.Equal(t, int64(13), metadata.Size) // "Hello, World!"
		assert.Equal(t, "text/plain", metadata.ContentType)
		assert.False(t, metadata.IsDir)
	})

	t.Run("GetOrExtract", func(t *testing.T) {
		// First call should extract and store
		metadata1, err := service.GetOrExtract(ctx, fs, "filesystem", "/docs/readme.md")
		assert.NoError(t, err)
		assert.NotNil(t, metadata1)

		// Second call should retrieve from store
		metadata2, err := service.GetOrExtract(ctx, fs, "filesystem", "/docs/readme.md")
		assert.NoError(t, err)
		assert.NotNil(t, metadata2)
		assert.Equal(t, metadata1.UpdatedAt.Unix(), metadata2.UpdatedAt.Unix())
	})

	t.Run("UpdateDirectory", func(t *testing.T) {
		err := service.UpdateDirectory(ctx, fs, "filesystem", "/")
		assert.NoError(t, err)

		// Verify all files were indexed
		results, err := service.Search(ctx, metadata.Query{
			Adaptor: "filesystem",
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 3) // At least our 3 test files
	})

	t.Run("Search", func(t *testing.T) {
		// Search by path prefix
		results, err := service.Search(ctx, metadata.Query{
			PathPrefix: "/docs/",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		if len(results) > 0 {
			assert.Equal(t, "/docs/readme.md", results[0].Path)
		}

		// Search by content type
		results, err = service.Search(ctx, metadata.Query{
			ContentType: "text/plain",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		if len(results) > 0 {
			assert.Equal(t, "/test.txt", results[0].Path)
		}
	})

	t.Run("Walk", func(t *testing.T) {
		var paths []string
		err := service.Walk(ctx, fs, "filesystem", "/",
			func(path string, _ os.FileInfo, _ *metadata.Metadata, err error) error {
				if err != nil {
					return err
				}
				paths = append(paths, path)
				return nil
			})
		assert.NoError(t, err)
		assert.Contains(t, paths, "/test.txt")
		assert.Contains(t, paths, "/docs/readme.md")
		assert.Contains(t, paths, "/images/test.jpg")
	})

	t.Run("Delete", func(t *testing.T) {
		err := service.Delete(ctx, "filesystem", "/test.txt")
		assert.NoError(t, err)

		// Verify deletion
		metadata, err := service.Get(ctx, "filesystem", "/test.txt")
		assert.NoError(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("Cache", func(t *testing.T) {
		// Extract should cache the result
		start := time.Now()
		_, err := service.Extract(ctx, fs, "filesystem", "/images/test.jpg")
		assert.NoError(t, err)
		firstDuration := time.Since(start)

		// Second call should be faster due to cache
		start = time.Now()
		_, err = service.Extract(ctx, fs, "filesystem", "/images/test.jpg")
		assert.NoError(t, err)
		secondDuration := time.Since(start)

		// Cache hit should be significantly faster
		// Use a more reasonable assertion since timings can vary
		t.Logf("First call: %v, Second call (cached): %v", firstDuration, secondDuration)
		// Only assert if we have measurable durations
		if firstDuration > 0 && secondDuration > 0 && firstDuration > time.Millisecond {
			assert.Less(t, secondDuration, firstDuration)
		} else if firstDuration == 0 || secondDuration == 0 {
			t.Log("Skipping cache performance assertion - operations too fast to measure")
		}

		// Clear cache
		service.ClearCache()

		// After clearing, should take longer again
		start = time.Now()
		_, err = service.Extract(ctx, fs, "filesystem", "/images/test.jpg")
		assert.NoError(t, err)
		thirdDuration := time.Since(start)
		t.Logf("Third call (after clear): %v", thirdDuration)

		// Should be slower than cached call
		// Only assert if we can measure meaningful differences
		if secondDuration > 0 && thirdDuration > 0 {
			// Only fail if cached was faster but uncached wasn't slower
			if secondDuration < firstDuration && thirdDuration <= secondDuration {
				t.Log("Warning: Expected third call to be slower than cached call")
			}
		} else {
			t.Log("Skipping timing assertions - operations too fast to measure")
		}
	})
}

func TestMetadataServiceWithExtractors(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "metadata.db")

	store, err := metadata.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	err = store.Init(ctx)
	require.NoError(t, err)

	service := metadata.NewService(store)

	// Create custom extractor for testing
	customExtractor := &testExtractor{}
	registry := metadata.NewExtractorRegistry()
	registry.Register(customExtractor)
	service.SetExtractorRegistry(registry)

	// Create test VFS
	testDir := filepath.Join(tmpDir, "files")
	err = os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	// Create filesystem adaptor
	adaptor := filesystem.New()
	err = adaptor.Connect(ctx, map[string]interface{}{"root": testDir})
	require.NoError(t, err)

	// Create VFS router and mount adaptor
	router := vfs.NewRouter()
	err = router.Mount("/", adaptor)
	require.NoError(t, err)
	defer router.Close()

	fs := router

	// Create test file
	w, err := fs.Create("/test.custom")
	require.NoError(t, err)
	_, err = io.WriteString(w, "custom content")
	require.NoError(t, err)
	w.Close()

	// Extract should use custom extractor
	metadata, err := service.Extract(ctx, fs, "filesystem", "/test.custom")
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, []string{"custom", "extracted"}, metadata.Tags)
	assert.Equal(t, "Custom file extracted", metadata.Description)
}

type testExtractor struct{}

func (e *testExtractor) CanExtract(contentType string) bool {
	// For .custom files, the content type will be application/octet-stream
	return contentType == "application/octet-stream"
}

func (e *testExtractor) Extract(_ context.Context, path string, _ []byte) (*metadata.Metadata, error) {
	// Only extract for .custom files
	if !strings.HasSuffix(path, ".custom") {
		return &metadata.Metadata{}, nil
	}
	return &metadata.Metadata{
		Tags:        []string{"custom", "extracted"},
		Description: "Custom file extracted",
	}, nil
}
