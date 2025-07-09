package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/px4n/finspect/adaptors/filesystem"
	"github.com/px4n/finspect/pkg/metadata"
	"github.com/px4n/finspect/pkg/search"
	"github.com/px4n/finspect/pkg/vfs"
	"github.com/spf13/cobra"
)

const (
	jsonFormat        = "json"
	filesystemAdaptor = "filesystem"
)

var (
	metadataDBPath    string
	metadataIndexPath string
	metadataFormat    string
	useFullTextSearch bool
)

func newMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Metadata operations",
		Long:  "Extract, store, and search file metadata",
	}

	cmd.PersistentFlags().StringVar(&metadataDBPath, "db", "finspect.db", "metadata database path")
	cmd.PersistentFlags().StringVar(&metadataIndexPath, "index", "finspect.bleve", "search index path")

	cmd.AddCommand(newMetadataExtractCmd())
	cmd.AddCommand(newMetadataShowCmd())
	cmd.AddCommand(newMetadataIndexCmd())
	cmd.AddCommand(newMetadataSearchCmd())

	return cmd
}

var metadataCmd = newMetadataCmd()

func newMetadataExtractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract [path]",
		Short: "Extract metadata from a file",
		Args:  cobra.ExactArgs(1),
		RunE:  runMetadataExtract,
	}

	cmd.Flags().StringVarP(&metadataFormat, "format", "f", "text", "output format (text, json)")
	cmd.Flags().BoolVar(&useFullTextSearch, "full-text", false, "use full-text search index")

	return cmd
}

func newMetadataShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [path]",
		Short: "Show stored metadata for a file",
		Args:  cobra.ExactArgs(1),
		RunE:  runMetadataShow,
	}

	cmd.Flags().StringVarP(&metadataFormat, "format", "f", "text", "output format (text, json)")

	return cmd
}

func newMetadataIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index [directory]",
		Short: "Index all files in a directory",
		Args:  cobra.ExactArgs(1),
		RunE:  runMetadataIndex,
	}

	cmd.Flags().BoolVar(&useFullTextSearch, "full-text", false, "use full-text search index")

	return cmd
}

func newMetadataSearchCmd() *cobra.Command {
	var (
		searchText        string
		searchContentType string
		searchTags        []string
		searchMinSize     int64
		searchMaxSize     int64
		searchLimit       int
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search metadata",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMetadataSearch(searchText, searchContentType, searchTags, searchMinSize, searchMaxSize, searchLimit)
		},
	}

	cmd.Flags().StringVarP(&searchText, "text", "t", "", "search text")
	cmd.Flags().StringVarP(&searchContentType, "type", "c", "", "content type filter")
	cmd.Flags().StringSliceVar(&searchTags, "tags", nil, "tags filter")
	cmd.Flags().Int64Var(&searchMinSize, "min-size", 0, "minimum file size")
	cmd.Flags().Int64Var(&searchMaxSize, "max-size", 0, "maximum file size")
	cmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "maximum results")
	cmd.Flags().StringVarP(&metadataFormat, "format", "f", "text", "output format (text, json)")
	cmd.Flags().BoolVar(&useFullTextSearch, "full-text", false, "use full-text search with Bleve")

	return cmd
}

func runMetadataExtract(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get VFS from current path
	vfs, adaptorName, vfsPath, vfsErr := getVFSForPath(args[0])
	if vfsErr != nil {
		return fmt.Errorf("get VFS: %w", vfsErr)
	}

	// Create metadata store
	store, storeErr := metadata.NewSQLiteStore(metadataDBPath)
	if storeErr != nil {
		return fmt.Errorf("create metadata store: %w", storeErr)
	}
	defer store.Close()

	if initErr := store.Init(ctx); initErr != nil {
		return fmt.Errorf("init metadata store: %w", initErr)
	}

	// Create service with optional search index
	var service *metadata.Service
	if useFullTextSearch {
		index, indexErr := search.NewBleveIndex(metadataIndexPath)
		if indexErr != nil {
			return fmt.Errorf("create search index: %w", indexErr)
		}
		defer index.Close()
		service = metadata.NewServiceWithIndex(store, index)
	} else {
		service = metadata.NewService(store)
	}

	// Extract metadata
	meta, err := service.Extract(ctx, vfs, adaptorName, vfsPath)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	// Store it
	if err := service.Store(ctx, meta); err != nil {
		return fmt.Errorf("store metadata: %w", err)
	}

	// Display result
	displayMetadata(meta)
	return nil
}

func runMetadataShow(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get VFS info
	_, adaptorName, vfsPath, err := getVFSForPath(args[0])
	if err != nil {
		return fmt.Errorf("get VFS: %w", err)
	}

	// Create metadata store
	store, err := metadata.NewSQLiteStore(metadataDBPath)
	if err != nil {
		return fmt.Errorf("create metadata store: %w", err)
	}
	defer store.Close()

	// Get metadata
	meta, err := store.Get(ctx, adaptorName, vfsPath)
	if err != nil {
		return fmt.Errorf("get metadata: %w", err)
	}

	if meta == nil {
		fmt.Println("No metadata found for this file")
		return nil
	}

	// Display result
	displayMetadata(meta)
	return nil
}

func runMetadataIndex(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get VFS from current path
	vfs, adaptorName, vfsPath, vfsErr := getVFSForPath(args[0])
	if vfsErr != nil {
		return fmt.Errorf("get VFS: %w", vfsErr)
	}

	// Create metadata store
	store, storeErr := metadata.NewSQLiteStore(metadataDBPath)
	if storeErr != nil {
		return fmt.Errorf("create metadata store: %w", storeErr)
	}
	defer store.Close()

	if initErr := store.Init(ctx); initErr != nil {
		return fmt.Errorf("init metadata store: %w", initErr)
	}

	// Create service with optional search index
	var service *metadata.Service
	var index search.Indexer
	if useFullTextSearch {
		var indexErr error
		index, indexErr = search.NewBleveIndex(metadataIndexPath)
		if indexErr != nil {
			return fmt.Errorf("create search index: %w", indexErr)
		}
		defer index.Close()
		service = metadata.NewServiceWithIndex(store, index)
	} else {
		service = metadata.NewService(store)
	}

	// Index directory
	fmt.Printf("Indexing %s...\n", args[0])
	if useFullTextSearch {
		fmt.Printf("Using full-text search index at %s\n", metadataIndexPath)
	}

	count := 0
	err := service.Walk(ctx, vfs, adaptorName, vfsPath,
		func(path string, _ os.FileInfo, _ *metadata.Metadata, err error) error {
			if err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
				return nil // Continue walking
			}

			count++
			if count%100 == 0 {
				fmt.Printf("Indexed %d files...\n", count)
			}

			return nil
		})

	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	fmt.Printf("Successfully indexed %d files\n", count)

	// Show index stats if using full-text search
	if useFullTextSearch && index != nil {
		stats, err := index.GetStats()
		if err == nil {
			fmt.Printf("\nIndex Statistics:\n")
			fmt.Printf("  Documents: %d\n", stats.DocumentCount)
			fmt.Printf("  Index Size: %.2f MB\n", float64(stats.IndexSize)/(1024*1024))
		}
	}

	return nil
}

func runMetadataSearch(
	searchText, searchContentType string,
	searchTags []string,
	searchMinSize, searchMaxSize int64,
	searchLimit int,
) error {
	ctx := context.Background()

	if useFullTextSearch {
		return runFullTextSearch(ctx, searchText, searchContentType, searchTags, searchMinSize, searchMaxSize, searchLimit)
	}

	return runDatabaseSearch(ctx, searchText, searchContentType, searchTags, searchMinSize, searchMaxSize, searchLimit)
}

func runFullTextSearch(
	ctx context.Context,
	searchText, searchContentType string,
	searchTags []string,
	searchMinSize, searchMaxSize int64,
	searchLimit int,
) error {
	index, err := search.NewBleveIndex(metadataIndexPath)
	if err != nil {
		return fmt.Errorf("open search index: %w", err)
	}
	defer index.Close()

	options := buildSearchOptions(searchContentType, searchTags, searchMinSize, searchMaxSize, searchLimit)

	result, err := index.Search(ctx, searchText, options...)
	if err != nil {
		return fmt.Errorf("full-text search: %w", err)
	}

	return displayFullTextResults(result)
}

func runDatabaseSearch(
	ctx context.Context,
	searchText, searchContentType string,
	searchTags []string,
	searchMinSize, searchMaxSize int64,
	searchLimit int,
) error {
	store, err := metadata.NewSQLiteStore(metadataDBPath)
	if err != nil {
		return fmt.Errorf("create metadata store: %w", err)
	}
	defer store.Close()

	query := metadata.Query{
		Text:        searchText,
		ContentType: searchContentType,
		Tags:        searchTags,
		MinSize:     searchMinSize,
		MaxSize:     searchMaxSize,
		Limit:       searchLimit,
	}

	results, err := store.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("search metadata: %w", err)
	}

	return displayDatabaseResults(results)
}

func buildSearchOptions(contentType string, tags []string, minSize, maxSize int64, limit int) []search.Option {
	var options []search.Option
	options = append(options, search.WithLimit(limit))

	if contentType != "" {
		options = append(options, search.WithContentType(contentType))
	}
	if len(tags) > 0 {
		options = append(options, search.WithTags(tags...))
	}
	if minSize > 0 || maxSize > 0 {
		options = append(options, search.WithSizeRange(minSize, maxSize))
	}

	return options
}

func displayFullTextResults(result *search.Result) error {
	if metadataFormat == jsonFormat {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	if result.Total == 0 {
		fmt.Println("No results found")
		return nil
	}

	fmt.Printf("Found %d results (%.2fms):\n\n", result.Total, float64(result.Duration.Microseconds())/1000.0)
	for i, hit := range result.Hits {
		displaySearchHit(i+1, hit)
	}

	displayFacets(result.Facets)
	return nil
}

func displaySearchHit(index int, hit *search.Hit) {
	fmt.Printf("%d. %s (score: %.3f)\n", index, hit.Document.Path, hit.Score)
	fmt.Printf("   Adaptor: %s\n", hit.Document.Adaptor)
	fmt.Printf("   Size: %d bytes\n", hit.Document.Size)
	fmt.Printf("   Type: %s\n", hit.Document.ContentType)

	if len(hit.Highlights) > 0 {
		fmt.Println("   Highlights:")
		for field, highlights := range hit.Highlights {
			for _, highlight := range highlights {
				fmt.Printf("     %s: %s\n", field, highlight)
			}
		}
	}
	fmt.Println()
}

func displayFacets(facets map[string]*search.Facet) {
	if len(facets) == 0 {
		return
	}

	fmt.Println("\nFacets:")
	for name, facet := range facets {
		fmt.Printf("\n%s:\n", name)
		for _, term := range facet.Terms {
			fmt.Printf("  %s: %d\n", term.Term, term.Count)
		}
	}
}

func displayDatabaseResults(results []*metadata.Metadata) error {
	if metadataFormat == jsonFormat {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)
	}

	if len(results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	fmt.Printf("Found %d results:\n\n", len(results))
	for i, meta := range results {
		displayMetadataSearchResult(i+1, meta)
	}

	return nil
}

func displayMetadataSearchResult(index int, meta *metadata.Metadata) {
	fmt.Printf("%d. %s\n", index, meta.Path)
	fmt.Printf("   Adaptor: %s\n", meta.Adaptor)
	fmt.Printf("   Size: %d bytes\n", meta.Size)
	fmt.Printf("   Type: %s\n", meta.ContentType)
	if len(meta.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(meta.Tags, ", "))
	}
	if meta.Description != "" {
		fmt.Printf("   Description: %s\n", meta.Description)
	}
	fmt.Println()
}

func displayMetadata(meta *metadata.Metadata) {
	if metadataFormat == jsonFormat {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(meta)
		return
	}

	displayBasicInfo(meta)
	displayHashes(meta)
	displayOptionalFields(meta)
	displayProperties(meta)
	displayMediaInfo(meta)
	displayDocumentInfo(meta)
}

func displayBasicInfo(meta *metadata.Metadata) {
	fmt.Printf("Path: %s\n", meta.Path)
	fmt.Printf("Adaptor: %s\n", meta.Adaptor)
	fmt.Printf("Size: %d bytes\n", meta.Size)
	fmt.Printf("Modified: %s\n", meta.ModTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Type: %s\n", meta.ContentType)
	fmt.Printf("Is Directory: %v\n", meta.IsDir)
}

func displayHashes(meta *metadata.Metadata) {
	if meta.MD5 != "" {
		fmt.Printf("MD5: %s\n", meta.MD5)
	}
	if meta.SHA256 != "" {
		fmt.Printf("SHA256: %s\n", meta.SHA256)
	}
}

func displayOptionalFields(meta *metadata.Metadata) {
	if len(meta.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(meta.Tags, ", "))
	}
	if meta.Description != "" {
		fmt.Printf("Description: %s\n", meta.Description)
	}
}

func displayProperties(meta *metadata.Metadata) {
	if len(meta.Properties) > 0 {
		fmt.Println("Properties:")
		for k, v := range meta.Properties {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

func displayMediaInfo(meta *metadata.Metadata) {
	if meta.MediaInfo == nil {
		return
	}

	fmt.Println("Media Info:")
	if meta.MediaInfo.Width > 0 && meta.MediaInfo.Height > 0 {
		fmt.Printf("  Dimensions: %dx%d\n", meta.MediaInfo.Width, meta.MediaInfo.Height)
	}
	if meta.MediaInfo.Duration > 0 {
		fmt.Printf("  Duration: %.2f seconds\n", meta.MediaInfo.Duration)
	}
	if meta.MediaInfo.FrameRate > 0 {
		fmt.Printf("  Frame Rate: %.2f fps\n", meta.MediaInfo.FrameRate)
	}
	if meta.MediaInfo.Codec != "" {
		fmt.Printf("  Codec: %s\n", meta.MediaInfo.Codec)
	}
}

func displayDocumentInfo(meta *metadata.Metadata) {
	if meta.DocumentInfo == nil {
		return
	}

	fmt.Println("Document Info:")
	if meta.DocumentInfo.PageCount > 0 {
		fmt.Printf("  Pages: %d\n", meta.DocumentInfo.PageCount)
	}
	if meta.DocumentInfo.WordCount > 0 {
		fmt.Printf("  Words: %d\n", meta.DocumentInfo.WordCount)
	}
	if meta.DocumentInfo.Author != "" {
		fmt.Printf("  Author: %s\n", meta.DocumentInfo.Author)
	}
	if meta.DocumentInfo.Title != "" {
		fmt.Printf("  Title: %s\n", meta.DocumentInfo.Title)
	}
}

// getVFSForPath returns the VFS, adaptor name, and VFS path for a given path
func getVFSForPath(path string) (vfs.VFS, string, string, error) {
	// Simple approach: use a temporary filesystem adaptor
	ctx := context.Background()

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", "", err
	}

	// Create a simple VFS with filesystem mount
	vfsInstance := vfs.NewRouter()
	adaptor := filesystem.New()

	// Mount at current directory
	cwd, _ := os.Getwd()
	if connectErr := adaptor.Connect(ctx, map[string]interface{}{"root": cwd}); connectErr != nil {
		return nil, "", "", fmt.Errorf("connect filesystem adaptor: %w", connectErr)
	}
	if mountErr := vfsInstance.Mount("/", adaptor); mountErr != nil {
		return nil, "", "", fmt.Errorf("mount filesystem: %w", mountErr)
	}

	// Convert to VFS path
	relPath, relErr := filepath.Rel(cwd, absPath)
	if relErr != nil {
		return nil, "", "", fmt.Errorf("make relative path: %w", relErr)
	}

	vfsPath := "/" + filepath.ToSlash(relPath)
	if relPath == "." {
		vfsPath = "/"
	}

	return vfsInstance, filesystemAdaptor, vfsPath, nil
}
