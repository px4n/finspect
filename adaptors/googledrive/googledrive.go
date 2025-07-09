package googledrive

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/px4n/finspect/adaptors/cloud"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Register registers the Google Drive provider
func Register() {
	cloud.RegisterProvider(cloud.ProviderGoogleDrive, New)
}

// Adaptor implements the CloudAdaptor interface for Google Drive
type Adaptor struct {
	*cloud.BaseCloudAdaptor
	service *drive.Service
	rootID  string // Root folder ID
}

// New creates a new Google Drive adaptor
func New() cloud.Adaptor {
	return &Adaptor{
		BaseCloudAdaptor: cloud.NewBaseCloudAdaptor(cloud.ProviderGoogleDrive),
	}
}

// Connect establishes connection to Google Drive
func (a *Adaptor) Connect(ctx context.Context, cfg cloud.AdaptorConfig) error {
	// Build OAuth2 config
	var config *oauth2.Config
	var token *oauth2.Token

	if cfg.Credentials != nil {
		// Extract OAuth2 credentials
		if clientID, ok := cfg.Credentials["client_id"].(string); ok {
			if clientSecret, ok := cfg.Credentials["client_secret"].(string); ok {
				config = &oauth2.Config{
					ClientID:     clientID,
					ClientSecret: clientSecret,
					Endpoint:     google.Endpoint,
					Scopes:       []string{drive.DriveScope},
				}
			}
		}

		// Extract token
		if tokenStr, ok := cfg.Credentials["refresh_token"].(string); ok {
			token = &oauth2.Token{
				RefreshToken: tokenStr,
			}
		} else if accessToken, ok := cfg.Credentials["access_token"].(string); ok {
			token = &oauth2.Token{
				AccessToken: accessToken,
			}
		}
	}

	// Alternative: use service account key
	if cfg.Credentials != nil {
		if keyFile, ok := cfg.Credentials["service_account_key"].(string); ok {
			service, err := drive.NewService(ctx, option.WithCredentialsFile(keyFile))
			if err != nil {
				return fmt.Errorf("create drive service with key file: %w", err)
			}
			a.service = service
			a.rootID = "root"
			return nil
		}
	}

	// Create client with OAuth2
	if config != nil && token != nil {
		client := config.Client(ctx, token)
		service, err := drive.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("create drive service: %w", err)
		}
		a.service = service
		a.rootID = "root"
		return nil
	}

	return fmt.Errorf("no valid credentials provided")
}

// Disconnect closes the connection
func (a *Adaptor) Disconnect(_ context.Context) error {
	a.service = nil
	return nil
}

// Stat returns information about a file or directory
func (a *Adaptor) Stat(ctx context.Context, path string) (*cloud.FileInfo, error) {
	fileID, err := a.getFileID(ctx, path)
	if err != nil {
		return nil, err
	}

	file, err := a.service.Files.Get(fileID).
		Fields("id, name, size, modifiedTime, mimeType, parents").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}

	return a.fileToCloudInfo(file, path), nil
}

// List returns a list of files in a directory
func (a *Adaptor) List(ctx context.Context, path string, options cloud.ListOptions) ([]*cloud.FileInfo, error) {
	parentID, err := a.getFileID(ctx, path)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("'%s' in parents and trashed = false", parentID)

	var files []*cloud.FileInfo
	pageToken := ""

	for {
		call := a.service.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, size, modifiedTime, mimeType, parents)").
			Context(ctx)

		if options.MaxKeys > 0 {
			call = call.PageSize(int64(options.MaxKeys))
		}

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list files: %w", err)
		}

		for _, file := range resp.Files {
			cloudPath := path + "/" + file.Name
			if path == "/" {
				cloudPath = "/" + file.Name
			}
			files = append(files, a.fileToCloudInfo(file, cloudPath))
		}

		pageToken = resp.NextPageToken
		if pageToken == "" || (options.MaxKeys > 0 && len(files) >= options.MaxKeys) {
			break
		}
	}

	return files, nil
}

// Read downloads a file from Google Drive
func (a *Adaptor) Read(ctx context.Context, path string, _ cloud.DownloadOptions) (io.ReadCloser, error) {
	fileID, err := a.getFileID(ctx, path)
	if err != nil {
		return nil, err
	}

	// Check if it's a Google Docs file that needs export
	file, err := a.service.Files.Get(fileID).Fields("mimeType").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("get file info: %w", err)
	}

	// Export Google Docs files
	if strings.HasPrefix(file.MimeType, "application/vnd.google-apps.") {
		exportMimeType := getExportMimeType(file.MimeType)
		resp, exportErr := a.service.Files.Export(fileID, exportMimeType).Context(ctx).Download()
		if exportErr != nil {
			return nil, fmt.Errorf("export file: %w", exportErr)
		}
		return resp.Body, nil
	}

	// Download regular files
	resp, err := a.service.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	return resp.Body, nil
}

// Write uploads a file to Google Drive
func (a *Adaptor) Write(ctx context.Context, path string, reader io.Reader, options cloud.UploadOptions) error {
	dir, name := splitPath(path)

	// Get parent folder ID
	parentID := a.rootID
	if dir != "/" && dir != "" {
		var err error
		parentID, err = a.getOrCreateFolder(ctx, dir)
		if err != nil {
			return fmt.Errorf("get parent folder: %w", err)
		}
	}

	// Check if file exists
	existingID, _ := a.getFileID(ctx, path)

	file := &drive.File{
		Name:     name,
		Parents:  []string{parentID},
		MimeType: options.ContentType,
	}

	if existingID != "" {
		// Update existing file
		_, err := a.service.Files.Update(existingID, file).
			Media(reader).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("update file: %w", err)
		}
	} else {
		// Create new file
		_, err := a.service.Files.Create(file).
			Media(reader).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
	}

	return nil
}

// Delete removes a file from Google Drive
func (a *Adaptor) Delete(ctx context.Context, path string) error {
	fileID, err := a.getFileID(ctx, path)
	if err != nil {
		return err
	}

	err = a.service.Files.Delete(fileID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

// Move moves a file within Google Drive
func (a *Adaptor) Move(ctx context.Context, oldPath, newPath string) error {
	fileID, err := a.getFileID(ctx, oldPath)
	if err != nil {
		return err
	}

	// Get current file info
	file, err := a.service.Files.Get(fileID).Fields("parents").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}

	// Get new parent
	newDir, newName := splitPath(newPath)
	newParentID := a.rootID
	if newDir != "/" && newDir != "" {
		newParentID, err = a.getOrCreateFolder(ctx, newDir)
		if err != nil {
			return fmt.Errorf("get new parent: %w", err)
		}
	}

	// Update file
	update := &drive.File{
		Name: newName,
	}

	_, err = a.service.Files.Update(fileID, update).
		AddParents(newParentID).
		RemoveParents(strings.Join(file.Parents, ",")).
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("move file: %w", err)
	}

	return nil
}

// Copy copies a file within Google Drive
func (a *Adaptor) Copy(ctx context.Context, srcPath, dstPath string) error {
	fileID, err := a.getFileID(ctx, srcPath)
	if err != nil {
		return err
	}

	// Get destination parent
	dstDir, dstName := splitPath(dstPath)
	dstParentID := a.rootID
	if dstDir != "/" && dstDir != "" {
		dstParentID, err = a.getOrCreateFolder(ctx, dstDir)
		if err != nil {
			return fmt.Errorf("get destination parent: %w", err)
		}
	}

	// Copy file
	copyFile := &drive.File{
		Name:    dstName,
		Parents: []string{dstParentID},
	}

	_, err = a.service.Files.Copy(fileID, copyFile).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return nil
}

// CreateDirectory creates a directory in Google Drive
func (a *Adaptor) CreateDirectory(ctx context.Context, path string) error {
	_, err := a.getOrCreateFolder(ctx, path)
	return err
}

// GetSignedURL generates a shareable link for temporary access
func (a *Adaptor) GetSignedURL(ctx context.Context, path string, _ time.Duration) (string, error) {
	fileID, err := a.getFileID(ctx, path)
	if err != nil {
		return "", err
	}

	// Create permission for anyone with link
	permission := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}

	_, err = a.service.Permissions.Create(fileID, permission).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("create permission: %w", err)
	}

	// Get the webViewLink
	file, err := a.service.Files.Get(fileID).Fields("webViewLink").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("get file link: %w", err)
	}

	// Note: Google Drive doesn't support time-limited public links through the API
	// The link will remain valid until the permission is revoked

	return file.WebViewLink, nil
}

// Helper methods

// getFileID gets the file ID from a path
func (a *Adaptor) getFileID(ctx context.Context, filePath string) (string, error) {
	if filePath == "/" || filePath == "" {
		return a.rootID, nil
	}

	parts := strings.Split(strings.Trim(filePath, "/"), "/")
	parentID := a.rootID

	for _, part := range parts {
		query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", part, parentID)
		resp, err := a.service.Files.List().
			Q(query).
			Fields("files(id)").
			Context(ctx).
			Do()
		if err != nil {
			return "", fmt.Errorf("search for %s: %w", part, err)
		}

		if len(resp.Files) == 0 {
			return "", fmt.Errorf("file not found: %s", filePath)
		}

		parentID = resp.Files[0].Id
	}

	return parentID, nil
}

// getOrCreateFolder gets or creates a folder
func (a *Adaptor) getOrCreateFolder(ctx context.Context, folderPath string) (string, error) {
	// Try to get existing folder
	folderID, err := a.getFileID(ctx, folderPath)
	if err == nil {
		return folderID, nil
	}

	// Create folder hierarchy
	parts := strings.Split(strings.Trim(folderPath, "/"), "/")
	parentID := a.rootID

	for _, part := range parts {
		// Check if folder exists
		query := fmt.Sprintf(
			"name = '%s' and '%s' in parents and mimeType = 'application/vnd.google-apps.folder' and trashed = false",
			part, parentID)
		resp, err := a.service.Files.List().
			Q(query).
			Fields("files(id)").
			Context(ctx).
			Do()
		if err != nil {
			return "", fmt.Errorf("search for folder %s: %w", part, err)
		}

		if len(resp.Files) > 0 {
			parentID = resp.Files[0].Id
		} else {
			// Create folder
			folder := &drive.File{
				Name:     part,
				MimeType: "application/vnd.google-apps.folder",
				Parents:  []string{parentID},
			}

			created, err := a.service.Files.Create(folder).Context(ctx).Do()
			if err != nil {
				return "", fmt.Errorf("create folder %s: %w", part, err)
			}

			parentID = created.Id
		}
	}

	return parentID, nil
}

// fileToCloudInfo converts a Google Drive file to FileInfo
func (a *Adaptor) fileToCloudInfo(file *drive.File, path string) *cloud.FileInfo {
	info := &cloud.FileInfo{
		Path:        path,
		Size:        file.Size,
		ContentType: file.MimeType,
		IsDir:       file.MimeType == "application/vnd.google-apps.folder",
		Metadata:    make(map[string]string),
	}

	if file.ModifiedTime != "" {
		if t, err := time.Parse(time.RFC3339, file.ModifiedTime); err == nil {
			info.ModTime = t
		}
	}

	info.Metadata["id"] = file.Id

	return info
}

// splitPath splits a path into directory and filename
func splitPath(p string) (dir, name string) {
	p = strings.Trim(p, "/")
	lastSlash := strings.LastIndex(p, "/")
	if lastSlash == -1 {
		return "/", p
	}
	return "/" + p[:lastSlash], p[lastSlash+1:]
}

const (
	mimeTypePDF = "application/pdf"
)

// getExportMimeType returns the export MIME type for Google Docs files
func getExportMimeType(googleMimeType string) string {
	switch googleMimeType {
	case "application/vnd.google-apps.document":
		return mimeTypePDF
	case "application/vnd.google-apps.spreadsheet":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "application/vnd.google-apps.presentation":
		return mimeTypePDF
	case "application/vnd.google-apps.drawing":
		return "image/png"
	default:
		return mimeTypePDF
	}
}
