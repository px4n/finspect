package pathutil_test

import (
	"testing"

	"github.com/px4n/finspect/internal/pathutil"
	"github.com/stretchr/testify/assert"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantMount    string
		wantRelative string
	}{
		{
			name:         "root path",
			path:         "/",
			wantMount:    "/",
			wantRelative: "/",
		},
		{
			name:         "mount point only",
			path:         "/gdrive",
			wantMount:    "/gdrive",
			wantRelative: "/",
		},
		{
			name:         "file in mount",
			path:         "/gdrive/file.txt",
			wantMount:    "/gdrive",
			wantRelative: "/file.txt",
		},
		{
			name:         "nested path",
			path:         "/local/Documents/Projects/file.go",
			wantMount:    "/local",
			wantRelative: "/Documents/Projects/file.go",
		},
		{
			name:         "path with dots",
			path:         "/mount/../other/file.txt",
			wantMount:    "/other",
			wantRelative: "/file.txt",
		},
		{
			name:         "relative path (invalid)",
			path:         "relative/path",
			wantMount:    "",
			wantRelative: "",
		},
		{
			name:         "empty path",
			path:         "",
			wantMount:    "",
			wantRelative: "",
		},
		{
			name:         "path with trailing slash",
			path:         "/mount/dir/",
			wantMount:    "/mount",
			wantRelative: "/dir",
		},
		{
			name:         "path with multiple slashes",
			path:         "/mount//dir///file.txt",
			wantMount:    "/mount",
			wantRelative: "/dir/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMount, gotRelative := pathutil.ParsePath(tt.path)
			assert.Equal(t, tt.wantMount, gotMount, "mount point mismatch")
			assert.Equal(t, tt.wantRelative, gotRelative, "relative path mismatch")
		})
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name     string
		elements []string
		want     string
	}{
		{
			name:     "empty elements",
			elements: []string{},
			want:     "",
		},
		{
			name:     "single element",
			elements: []string{"/mount"},
			want:     "/mount",
		},
		{
			name:     "multiple elements",
			elements: []string{"/mount", "dir", "file.txt"},
			want:     "/mount/dir/file.txt",
		},
		{
			name:     "elements with slashes",
			elements: []string{"/mount/", "/dir/", "file.txt"},
			want:     "/mount/dir/file.txt",
		},
		{
			name:     "elements with dots",
			elements: []string{"/mount", ".", "dir", "..", "file.txt"},
			want:     "/mount/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.Join(tt.elements...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantDir  string
		wantFile string
	}{
		{
			name:     "simple file",
			path:     "/mount/file.txt",
			wantDir:  "/mount/",
			wantFile: "file.txt",
		},
		{
			name:     "directory",
			path:     "/mount/dir/",
			wantDir:  "/mount/dir/",
			wantFile: "",
		},
		{
			name:     "root",
			path:     "/",
			wantDir:  "/",
			wantFile: "",
		},
		{
			name:     "file in root",
			path:     "/file.txt",
			wantDir:  "/",
			wantFile: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotFile := pathutil.Split(tt.path)
			assert.Equal(t, tt.wantDir, gotDir, "directory mismatch")
			assert.Equal(t, tt.wantFile, gotFile, "file mismatch")
		})
	}
}

func TestBase(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple file",
			path: "/mount/dir/file.txt",
			want: "file.txt",
		},
		{
			name: "directory",
			path: "/mount/dir/",
			want: "dir",
		},
		{
			name: "root",
			path: "/",
			want: "/",
		},
		{
			name: "dot",
			path: ".",
			want: ".",
		},
		{
			name: "empty",
			path: "",
			want: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.Base(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple file",
			path: "/mount/dir/file.txt",
			want: "/mount/dir",
		},
		{
			name: "directory",
			path: "/mount/dir/",
			want: "/mount/dir",
		},
		{
			name: "file in root",
			path: "/file.txt",
			want: "/",
		},
		{
			name: "root",
			path: "/",
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.Dir(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExt(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple extension",
			path: "file.txt",
			want: ".txt",
		},
		{
			name: "multiple dots",
			path: "archive.tar.gz",
			want: ".gz",
		},
		{
			name: "no extension",
			path: "file",
			want: "",
		},
		{
			name: "hidden file",
			path: ".gitignore",
			want: "",
		},
		{
			name: "path with extension",
			path: "/mount/dir/file.go",
			want: ".go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.Ext(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsMount(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "mount point",
			path: "/gdrive",
			want: true,
		},
		{
			name: "file in mount",
			path: "/gdrive/file.txt",
			want: false,
		},
		{
			name: "nested directory",
			path: "/local/Documents",
			want: false,
		},
		{
			name: "root",
			path: "/",
			want: true,
		},
		{
			name: "relative path",
			path: "relative",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.IsMount(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "already normalized",
			path: "/mount/dir/file.txt",
			want: "/mount/dir/file.txt",
		},
		{
			name: "relative path",
			path: "dir/file.txt",
			want: "/dir/file.txt",
		},
		{
			name: "path with dots",
			path: "/mount/./dir/../file.txt",
			want: "/mount/file.txt",
		},
		{
			name: "multiple slashes",
			path: "/mount//dir///file.txt",
			want: "/mount/dir/file.txt",
		},
		{
			name: "empty path",
			path: "",
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.NormalizePath(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRelativeTo(t *testing.T) {
	tests := []struct {
		name       string
		base       string
		target     string
		wantResult string
		wantErr    bool
	}{
		{
			name:       "simple relative",
			base:       "/mount",
			target:     "/mount/dir/file.txt",
			wantResult: "dir/file.txt",
			wantErr:    false,
		},
		{
			name:       "same path",
			base:       "/mount/dir",
			target:     "/mount/dir",
			wantResult: ".",
			wantErr:    false,
		},
		{
			name:       "not under base",
			base:       "/mount1",
			target:     "/mount2/file.txt",
			wantResult: "",
			wantErr:    false,
		},
		{
			name:       "nested base",
			base:       "/mount/a/b",
			target:     "/mount/a/b/c/d/file.txt",
			wantResult: "c/d/file.txt",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pathutil.RelativeTo(tt.base, tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, got)
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		want   bool
	}{
		{
			name:   "has prefix",
			path:   "/mount/dir/file.txt",
			prefix: "/mount",
			want:   true,
		},
		{
			name:   "exact match",
			path:   "/mount",
			prefix: "/mount",
			want:   true,
		},
		{
			name:   "no prefix",
			path:   "/other/file.txt",
			prefix: "/mount",
			want:   false,
		},
		{
			name:   "partial match not prefix",
			path:   "/mount2/file.txt",
			prefix: "/mount",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.HasPrefix(tt.path, tt.prefix)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTrimPrefix(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		want   string
	}{
		{
			name:   "trim prefix",
			path:   "/mount/dir/file.txt",
			prefix: "/mount",
			want:   "/dir/file.txt",
		},
		{
			name:   "exact match",
			path:   "/mount",
			prefix: "/mount",
			want:   "/",
		},
		{
			name:   "no prefix",
			path:   "/other/file.txt",
			prefix: "/mount",
			want:   "/other/file.txt",
		},
		{
			name:   "nested prefix",
			path:   "/mount/a/b/c",
			prefix: "/mount/a",
			want:   "/b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.TrimPrefix(tt.path, tt.prefix)
			assert.Equal(t, tt.want, got)
		})
	}
}
