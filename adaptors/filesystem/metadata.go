package filesystem

import (
	"crypto/md5" // #nosec G501 - MD5 used for file integrity, not security
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/px4n/finspect/pkg/vfs"
)

// Metadata represents file metadata
type Metadata struct {
	// Basic file info
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`

	// Hashes
	MD5    string `json:"md5,omitempty"`
	SHA256 string `json:"sha256,omitempty"`

	// File type info
	MimeType  string `json:"mime_type,omitempty"`
	Extension string `json:"extension,omitempty"`

	// System info
	UID int `json:"uid,omitempty"`
	GID int `json:"gid,omitempty"`

	// Additional attributes
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ExtractMetadata extracts metadata from a file
func (a *Adaptor) ExtractMetadata(path string) (*Metadata, error) {
	if !a.connected {
		return nil, vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)

	// Get file info
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, mapError(err)
	}

	meta := &Metadata{
		Name:       info.Name(),
		Path:       path,
		Size:       info.Size(),
		Mode:       info.Mode().String(),
		ModTime:    info.ModTime().Unix(),
		IsDir:      info.IsDir(),
		Extension:  strings.ToLower(filepath.Ext(info.Name())),
		Attributes: make(map[string]interface{}),
	}

	// Get system info if available
	if sysInfo := getSystemInfo(info); sysInfo != nil {
		meta.UID = sysInfo.UID
		meta.GID = sysInfo.GID
	}

	// Skip further processing for directories
	if info.IsDir() {
		return meta, nil
	}

	// Detect MIME type
	meta.MimeType = mime.TypeByExtension(meta.Extension)
	if meta.MimeType == "" {
		// Try to detect by reading file header
		if mimeType, err := detectMimeType(realPath); err == nil {
			meta.MimeType = mimeType
		}
	}

	// Calculate hashes for small files (< 100MB)
	if info.Size() < 100*1024*1024 {
		if err := a.calculateHashes(realPath, meta); err != nil {
			// Non-fatal: just log and continue
			meta.Attributes["hash_error"] = err.Error()
		}
	}

	// Extract additional metadata based on file type
	a.extractTypeSpecificMetadata(realPath, meta)

	return meta, nil
}

// calculateHashes calculates MD5 and SHA256 hashes for a file
func (a *Adaptor) calculateHashes(path string, meta *Metadata) error {
	file, err := os.Open(path) // #nosec G304 - path is already sanitized by resolvePath
	if err != nil {
		return err
	}
	defer file.Close()

	md5Hash := md5.New() // #nosec G401 - MD5 used for file integrity, not security
	sha256Hash := sha256.New()

	// Create a multi-writer to calculate both hashes in one pass
	writer := io.MultiWriter(md5Hash, sha256Hash)

	if _, err := io.Copy(writer, file); err != nil {
		return err
	}

	meta.MD5 = hex.EncodeToString(md5Hash.Sum(nil))
	meta.SHA256 = hex.EncodeToString(sha256Hash.Sum(nil))

	return nil
}

// detectMimeType tries to detect MIME type by reading file header
func detectMimeType(path string) (string, error) {
	file, err := os.Open(path) // #nosec G304 - path is already sanitized
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read first 512 bytes for detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	return detectMimeFromBytes(buffer[:n]), nil
}

// detectMimeFromBytes detects MIME type from file header bytes
func detectMimeFromBytes(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}

	// Check for image types
	if mimeType := detectImageType(data); mimeType != "" {
		return mimeType
	}

	// Check for document types
	if mimeType := detectDocumentType(data); mimeType != "" {
		return mimeType
	}

	// Check for text
	if isTextFile(data) {
		return "text/plain"
	}

	return "application/octet-stream"
}

// detectImageType checks if data represents an image file
func detectImageType(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// GIF
	if string(data[:3]) == "GIF" {
		return "image/gif"
	}

	// WebP
	if string(data[:4]) == "RIFF" && len(data) > 11 && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return ""
}

// detectDocumentType checks if data represents a document file
func detectDocumentType(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	// PDF
	if string(data[:4]) == "%PDF" {
		return "application/pdf"
	}

	// ZIP-based formats (docx, xlsx, etc.)
	if data[0] == 0x50 && data[1] == 0x4B && (data[2] == 0x03 || data[2] == 0x05 || data[2] == 0x07) {
		return "application/zip"
	}

	return ""
}

// isTextFile checks if the data appears to be text
func isTextFile(data []byte) bool {
	for _, b := range data {
		// Check for non-printable characters (excluding common whitespace)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return false
		}
		if b > 126 && b < 128 {
			return false
		}
	}
	return true
}

// extractTypeSpecificMetadata extracts metadata specific to certain file types
func (a *Adaptor) extractTypeSpecificMetadata(path string, meta *Metadata) {
	switch {
	case strings.HasPrefix(meta.MimeType, "image/"):
		// TODO: Extract image dimensions, EXIF data
		meta.Attributes["type"] = "image"

	case strings.HasPrefix(meta.MimeType, "video/"):
		// TODO: Extract video duration, resolution, codec
		meta.Attributes["type"] = "video"

	case strings.HasPrefix(meta.MimeType, "audio/"):
		// TODO: Extract audio duration, bitrate, tags
		meta.Attributes["type"] = "audio"

	case strings.HasPrefix(meta.MimeType, "text/"):
		// Extract line count for text files
		if lines, err := countLines(path); err == nil {
			meta.Attributes["lines"] = lines
		}
		meta.Attributes["type"] = "text"

	default:
		// Determine type by extension
		switch meta.Extension {
		case ".go", ".js", ".py", ".java", ".c", ".cpp", ".rs":
			meta.Attributes["type"] = "code"
		case ".json", ".yaml", ".yml", ".xml", ".toml":
			meta.Attributes["type"] = "config"
		case ".md", ".txt", ".rst":
			meta.Attributes["type"] = "document"
		default:
			meta.Attributes["type"] = "binary"
		}
	}
}

// countLines counts the number of lines in a text file
func countLines(path string) (int, error) {
	file, err := os.Open(path) // #nosec G304 - path is already sanitized
	if err != nil {
		return 0, err
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := file.Read(buf)
		count += len(strings.Split(string(buf[:c]), string(lineSep))) - 1

		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}
