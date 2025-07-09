package metadata_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/px4n/finspect/pkg/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStore(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := metadata.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	err = store.Init(ctx)
	require.NoError(t, err)

	t.Run("PutAndGet", func(t *testing.T) {
		// Create test metadata
		now := time.Now()
		meta := &metadata.Metadata{
			Path:        "/test/file.txt",
			Adaptor:     "filesystem",
			UpdatedAt:   now,
			Size:        1024,
			Mode:        0644,
			ModTime:     now,
			IsDir:       false,
			ContentType: "text/plain",
			MD5:         "d41d8cd98f00b204e9800998ecf8427e",
			SHA256:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Tags:        []string{"test", "document"},
			Description: "Test file",
			Properties: map[string]string{
				"author":  "test",
				"version": "1.0",
			},
		}

		// Put metadata
		err := store.Put(ctx, meta)
		assert.NoError(t, err)

		// Get metadata
		retrieved, err := store.Get(ctx, "filesystem", "/test/file.txt")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, meta.Path, retrieved.Path)
		assert.Equal(t, meta.Size, retrieved.Size)
		assert.Equal(t, meta.Tags, retrieved.Tags)
		assert.Equal(t, meta.Properties, retrieved.Properties)
	})

	t.Run("Update", func(t *testing.T) {
		// Update existing metadata
		meta := &metadata.Metadata{
			Path:        "/test/file.txt",
			Adaptor:     "filesystem",
			UpdatedAt:   time.Now(),
			Size:        2048, // Changed
			Mode:        0644,
			ModTime:     time.Now(),
			IsDir:       false,
			ContentType: "text/plain",
			Tags:        []string{"updated"},
			Description: "Updated file",
		}

		err := store.Put(ctx, meta)
		assert.NoError(t, err)

		// Verify update
		retrieved, err := store.Get(ctx, "filesystem", "/test/file.txt")
		assert.NoError(t, err)
		assert.Equal(t, int64(2048), retrieved.Size)
		assert.Equal(t, []string{"updated"}, retrieved.Tags)
		assert.Equal(t, "Updated file", retrieved.Description)
	})

	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(ctx, "filesystem", "/test/file.txt")
		assert.NoError(t, err)

		// Verify deletion
		retrieved, err := store.Get(ctx, "filesystem", "/test/file.txt")
		assert.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("Search", func(t *testing.T) {
		// Add test data
		testData := []*metadata.Metadata{
			{
				Path:        "/docs/report.pdf",
				Adaptor:     "filesystem",
				UpdatedAt:   time.Now(),
				Size:        5000,
				ContentType: "application/pdf",
				Tags:        []string{"report", "2024"},
				Description: "Annual report",
			},
			{
				Path:        "/docs/readme.txt",
				Adaptor:     "filesystem",
				UpdatedAt:   time.Now(),
				Size:        1000,
				ContentType: "text/plain",
				Tags:        []string{"documentation"},
				Description: "Project readme",
			},
			{
				Path:        "/images/photo.jpg",
				Adaptor:     "filesystem",
				UpdatedAt:   time.Now(),
				Size:        200000,
				ContentType: "image/jpeg",
				Tags:        []string{"photo", "2024"},
				MediaInfo: &metadata.MediaInfo{
					Width:  1920,
					Height: 1080,
				},
			},
		}

		for _, m := range testData {
			err := store.Put(ctx, m)
			require.NoError(t, err)
		}

		// Search by path prefix
		results, err := store.Search(ctx, metadata.Query{
			PathPrefix: "/docs/",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Search by content type
		results, err = store.Search(ctx, metadata.Query{
			ContentType: "text/plain",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "/docs/readme.txt", results[0].Path)

		// Search by size range
		results, err = store.Search(ctx, metadata.Query{
			MinSize: 1000,
			MaxSize: 10000,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Search by tags
		results, err = store.Search(ctx, metadata.Query{
			Tags: []string{"2024"},
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Full-text search
		results, err = store.Search(ctx, metadata.Query{
			Text: "report",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "/docs/report.pdf", results[0].Path)
	})

	t.Run("BatchOperations", func(t *testing.T) {
		// Batch put
		batch := []*metadata.Metadata{
			{
				Path:      "/batch/file1.txt",
				Adaptor:   "filesystem",
				UpdatedAt: time.Now(),
			},
			{
				Path:      "/batch/file2.txt",
				Adaptor:   "filesystem",
				UpdatedAt: time.Now(),
			},
			{
				Path:      "/batch/file3.txt",
				Adaptor:   "filesystem",
				UpdatedAt: time.Now(),
			},
		}

		err := store.BatchPut(ctx, batch)
		assert.NoError(t, err)

		// Verify all were added
		results, err := store.Search(ctx, metadata.Query{
			PathPrefix: "/batch/",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 3)

		// Batch delete
		paths := []metadata.PathKey{
			{Adaptor: "filesystem", Path: "/batch/file1.txt"},
			{Adaptor: "filesystem", Path: "/batch/file2.txt"},
		}
		err = store.BatchDelete(ctx, paths)
		assert.NoError(t, err)

		// Verify deletion
		results, err = store.Search(ctx, metadata.Query{
			PathPrefix: "/batch/",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "/batch/file3.txt", results[0].Path)
	})

	t.Run("MediaInfo", func(t *testing.T) {
		meta := &metadata.Metadata{
			Path:        "/media/video.mp4",
			Adaptor:     "filesystem",
			UpdatedAt:   time.Now(),
			ContentType: "video/mp4",
			MediaInfo: &metadata.MediaInfo{
				Width:     1920,
				Height:    1080,
				Duration:  120.5,
				FrameRate: 30.0,
				Codec:     "h264",
				Location: &metadata.GeoLocation{
					Latitude:  37.7749,
					Longitude: -122.4194,
				},
			},
		}

		err := store.Put(ctx, meta)
		assert.NoError(t, err)

		retrieved, err := store.Get(ctx, "filesystem", "/media/video.mp4")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved.MediaInfo)
		assert.Equal(t, 1920, retrieved.MediaInfo.Width)
		assert.Equal(t, 120.5, retrieved.MediaInfo.Duration)
		assert.NotNil(t, retrieved.MediaInfo.Location)
		assert.Equal(t, 37.7749, retrieved.MediaInfo.Location.Latitude)
	})

	t.Run("DocumentInfo", func(t *testing.T) {
		created := time.Now().Add(-24 * time.Hour)
		modified := time.Now()

		meta := &metadata.Metadata{
			Path:        "/docs/document.docx",
			Adaptor:     "filesystem",
			UpdatedAt:   time.Now(),
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			DocumentInfo: &metadata.DocumentInfo{
				WordCount:   1500,
				PageCount:   10,
				Author:      "John Doe",
				Title:       "Test Document",
				Keywords:    []string{"test", "document", "sample"},
				Created:     &created,
				Modified:    &modified,
				Application: "Microsoft Word",
			},
		}

		err := store.Put(ctx, meta)
		assert.NoError(t, err)

		retrieved, err := store.Get(ctx, "filesystem", "/docs/document.docx")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved.DocumentInfo)
		assert.Equal(t, 1500, retrieved.DocumentInfo.WordCount)
		assert.Equal(t, "John Doe", retrieved.DocumentInfo.Author)
		assert.Equal(t, []string{"test", "document", "sample"}, retrieved.DocumentInfo.Keywords)
	})
}

func TestSQLiteStoreConcurrency(t *testing.T) {
	// This test verifies that the SQLite store handles concurrent operations correctly
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_concurrent.db")

	store, err := metadata.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	err = store.Init(ctx)
	require.NoError(t, err)

	// Run concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			meta := &metadata.Metadata{
				Path:      fmt.Sprintf("/concurrent/file%d.txt", i),
				Adaptor:   "filesystem",
				UpdatedAt: time.Now(),
				Size:      int64(i * 100),
			}
			putErr := store.Put(ctx, meta)
			assert.NoError(t, putErr)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all were written
	results, err := store.Search(ctx, metadata.Query{
		PathPrefix: "/concurrent/",
	})
	assert.NoError(t, err)
	assert.Len(t, results, 10)
}
