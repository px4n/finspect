package googledrive

import (
	"context"
	"testing"

	"github.com/px4n/finspect/adaptors/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdaptorBasics(t *testing.T) {
	adaptor := New()
	assert.NotNil(t, adaptor)

	gd, ok := adaptor.(*Adaptor)
	require.True(t, ok)
	assert.Equal(t, cloud.ProviderGoogleDrive, gd.GetProvider())
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		wantDir  string
		wantName string
	}{
		{"/file.txt", "/", "file.txt"},
		{"/folder/file.txt", "/folder", "file.txt"},
		{"/a/b/c/file.txt", "/a/b/c", "file.txt"},
		{"file.txt", "/", "file.txt"},
		{"/", "/", ""},
		{"", "/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			dir, name := splitPath(tt.path)
			assert.Equal(t, tt.wantDir, dir)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestGetExportMimeType(t *testing.T) {
	tests := []struct {
		googleMimeType string
		want           string
	}{
		{"application/vnd.google-apps.document", "application/pdf"},
		{"application/vnd.google-apps.spreadsheet", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"application/vnd.google-apps.presentation", "application/pdf"},
		{"application/vnd.google-apps.drawing", "image/png"},
		{"application/unknown", "application/pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.googleMimeType, func(t *testing.T) {
			got := getExportMimeType(tt.googleMimeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// MockDriveService would be needed for more comprehensive testing
// This would mock the Google Drive API responses

func TestConnectErrors(t *testing.T) {
	ctx := context.Background()
	adaptor := New()

	// Test with no credentials
	err := adaptor.Connect(ctx, cloud.AdaptorConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid credentials")

	// Test with partial OAuth2 credentials
	err = adaptor.Connect(ctx, cloud.AdaptorConfig{
		Credentials: map[string]interface{}{
			"client_id": "test-id",
			// Missing client_secret
		},
	})
	assert.Error(t, err)
}

// Integration test example (requires actual Google Drive credentials)
func TestIntegrationGoogleDrive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This would require actual credentials to test
	// Example structure:
	/*
		ctx := context.Background()
		adaptor := New()

		cfg := cloud.CloudAdaptorConfig{
			Credentials: map[string]interface{}{
				"service_account_key": "/path/to/key.json",
			},
		}

		err := adaptor.Connect(ctx, cfg)
		require.NoError(t, err)
		defer adaptor.Disconnect(ctx)

		// Test operations...
	*/
}
