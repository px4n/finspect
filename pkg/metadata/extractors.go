package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"  // GIF image format
	_ "image/jpeg" // JPEG image format
	_ "image/png"  // PNG image format
	"strings"
)

// BasicExtractor extracts basic metadata from common file types.
type BasicExtractor struct{}

// NewBasicExtractor creates a new basic metadata extractor.
func NewBasicExtractor() *BasicExtractor {
	return &BasicExtractor{}
}

// CanExtract checks if this extractor can handle the file type.
func (e *BasicExtractor) CanExtract(_ string) bool {
	return true // Basic extractor handles all files
}

// Extract extracts basic metadata from a file.
func (e *BasicExtractor) Extract(_ context.Context, path string, _ []byte) (*Metadata, error) {
	metadata := &Metadata{
		Path: path,
	}

	// Extract tags from path
	parts := strings.Split(path, "/")
	if len(parts) > 2 {
		// Use directory names as tags
		for i := 1; i < len(parts)-1; i++ {
			if parts[i] != "" {
				metadata.Tags = append(metadata.Tags, parts[i])
			}
		}
	}

	return metadata, nil
}

// ImageExtractor extracts metadata from image files.
type ImageExtractor struct{}

// NewImageExtractor creates a new image metadata extractor.
func NewImageExtractor() *ImageExtractor {
	return &ImageExtractor{}
}

// CanExtract checks if this extractor can handle the file type.
func (e *ImageExtractor) CanExtract(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// Extract extracts metadata from an image file.
func (e *ImageExtractor) Extract(_ context.Context, path string, content []byte) (*Metadata, error) {
	// Decode image to get dimensions
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{
		Path: path,
		MediaInfo: &MediaInfo{
			Width:  config.Width,
			Height: config.Height,
		},
		Properties: map[string]string{
			"format": format,
		},
	}

	return metadata, nil
}

// TextExtractor extracts metadata from text files.
type TextExtractor struct{}

// NewTextExtractor creates a new text metadata extractor.
func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

// CanExtract checks if this extractor can handle the file type.
func (e *TextExtractor) CanExtract(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/x-yaml"
}

// Extract extracts metadata from a text file.
func (e *TextExtractor) Extract(_ context.Context, path string, content []byte) (*Metadata, error) {
	text := string(content)

	// Count words (simple implementation)
	words := strings.Fields(text)
	wordCount := len(words)

	// Count lines
	lines := strings.Split(text, "\n")
	lineCount := len(lines)

	metadata := &Metadata{
		Path: path,
		DocumentInfo: &DocumentInfo{
			WordCount: wordCount,
		},
		Properties: map[string]string{
			"lines": fmt.Sprintf("%d", lineCount),
		},
	}

	// Try to extract JSON metadata if applicable
	if strings.HasSuffix(path, ".json") {
		var jsonData map[string]interface{}
		if err := json.Unmarshal(content, &jsonData); err == nil {
			// Extract some common fields if present
			if title, ok := jsonData["title"].(string); ok {
				metadata.DocumentInfo.Title = title
			}
			if author, ok := jsonData["author"].(string); ok {
				metadata.DocumentInfo.Author = author
			}
			if description, ok := jsonData["description"].(string); ok {
				metadata.Description = description
			}
		}
	}

	return metadata, nil
}

// AudioExtractor extracts metadata from audio files.
type AudioExtractor struct{}

// NewAudioExtractor creates a new audio metadata extractor.
func NewAudioExtractor() *AudioExtractor {
	return &AudioExtractor{}
}

// CanExtract checks if this extractor can handle the file type.
func (e *AudioExtractor) CanExtract(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/")
}

// Extract extracts metadata from an audio file.
// Note: This is a placeholder implementation. In a real system,
// you would use a library like taglib or similar to extract metadata.
func (e *AudioExtractor) Extract(_ context.Context, path string, _ []byte) (*Metadata, error) {
	metadata := &Metadata{
		Path: path,
		MediaInfo: &MediaInfo{
			// Placeholder values - real implementation would extract from file
			Codec: "unknown",
		},
	}

	// Extract from filename if possible
	filename := path[strings.LastIndex(path, "/")+1:]
	parts := strings.Split(filename, " - ")
	if len(parts) >= 2 {
		metadata.MediaInfo.Artist = parts[0]
		metadata.MediaInfo.Title = strings.TrimSuffix(parts[1], ".mp3")
	}

	return metadata, nil
}

// VideoExtractor extracts metadata from video files.
type VideoExtractor struct{}

// NewVideoExtractor creates a new video metadata extractor.
func NewVideoExtractor() *VideoExtractor {
	return &VideoExtractor{}
}

// CanExtract checks if this extractor can handle the file type.
func (e *VideoExtractor) CanExtract(contentType string) bool {
	return strings.HasPrefix(contentType, "video/")
}

// Extract extracts metadata from a video file.
// Note: This is a placeholder implementation. In a real system,
// you would use a library like ffprobe or similar to extract metadata.
func (e *VideoExtractor) Extract(_ context.Context, path string, _ []byte) (*Metadata, error) {
	metadata := &Metadata{
		Path: path,
		MediaInfo: &MediaInfo{
			// Placeholder values - real implementation would extract from file
			Codec: "unknown",
		},
	}

	return metadata, nil
}

// NewDefaultExtractorRegistry creates a registry with default extractors.
func NewDefaultExtractorRegistry() *ExtractorRegistry {
	registry := NewExtractorRegistry()

	// Register extractors in order of specificity
	registry.Register(NewImageExtractor())
	registry.Register(NewTextExtractor())
	registry.Register(NewAudioExtractor())
	registry.Register(NewVideoExtractor())
	registry.Register(NewBasicExtractor()) // Catch-all

	return registry
}
