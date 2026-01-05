package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidaniva/immich-importer/internal/state"
)

// Importer handles uploading photos to Immich
type Importer struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
}

// ProgressCallback is called with progress updates
type ProgressCallback func(phase string, current, total int, currentFile string)

// New creates a new Importer
func New(serverURL, apiKey string) *Importer {
	return &Importer{
		serverURL: serverURL,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for uploads
		},
	}
}

// ImportFiles imports all downloaded files to Immich
func (i *Importer) ImportFiles(ctx context.Context, jobState *state.JobState, progress ProgressCallback) error {
	// Initialize upload state if needed
	if jobState.UploadState == nil {
		jobState.UploadState = &state.UploadState{
			UploadedFiles: []string{},
		}
	}

	// Create a set of already uploaded files
	uploadedSet := make(map[string]bool)
	for _, f := range jobState.UploadState.UploadedFiles {
		uploadedSet[f] = true
	}

	// Process each downloaded file
	for _, file := range jobState.Files {
		if !file.Downloaded || file.LocalPath == "" {
			continue
		}

		// Check if it's a zip file
		if strings.HasSuffix(strings.ToLower(file.Name), ".zip") {
			if err := i.processZipFile(ctx, file.LocalPath, jobState, uploadedSet, progress); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *Importer) processZipFile(ctx context.Context, zipPath string, jobState *state.JobState, uploadedSet map[string]bool, progress ProgressCallback) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer reader.Close()

	// Count total files
	var photoFiles []*zip.File
	for _, f := range reader.File {
		if !f.FileInfo().IsDir() && isMediaFile(f.Name) {
			photoFiles = append(photoFiles, f)
		}
	}

	jobState.UploadState.TotalPhotos += len(photoFiles)

	// Process each file
	for _, f := range photoFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Generate unique ID for this file
		fileID := fmt.Sprintf("%s:%s", zipPath, f.Name)
		if uploadedSet[fileID] {
			continue // Already uploaded
		}

		progress("uploading", jobState.UploadState.UploadedPhotos, jobState.UploadState.TotalPhotos, f.Name)

		// Extract and upload
		if err := i.uploadZipEntry(ctx, f); err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to upload %s: %v\n", f.Name, err)
			continue
		}

		// Mark as uploaded
		uploadedSet[fileID] = true
		jobState.UploadState.UploadedFiles = append(jobState.UploadState.UploadedFiles, fileID)
		jobState.UploadState.UploadedPhotos++

		// Save state periodically
		if jobState.UploadState.UploadedPhotos%100 == 0 {
			jobState.Save()
		}
	}

	return nil
}

func (i *Importer) uploadZipEntry(ctx context.Context, f *zip.File) error {
	// Open file in archive
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Read file content
	content, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	// Get modification time
	modTime := f.Modified
	if modTime.IsZero() {
		modTime = time.Now()
	}

	// Upload to Immich
	return i.uploadAsset(ctx, filepath.Base(f.Name), content, modTime)
}

func (i *Importer) uploadAsset(ctx context.Context, filename string, content []byte, modTime time.Time) error {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add asset data
	part, err := writer.CreateFormFile("assetData", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(content); err != nil {
		return err
	}

	// Add device asset ID (for deduplication)
	deviceAssetID := fmt.Sprintf("import-%x", content[:min(32, len(content))])
	writer.WriteField("deviceAssetId", deviceAssetID)
	writer.WriteField("deviceId", "immich-importer")
	writer.WriteField("fileCreatedAt", modTime.Format(time.RFC3339))
	writer.WriteField("fileModifiedAt", modTime.Format(time.RFC3339))

	writer.Close()

	// Create request
	url := fmt.Sprintf("%s/api/assets", i.serverURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-api-key", i.apiKey)

	// Send request
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err == nil {
			if msg, ok := errResp["message"].(string); ok {
				// Duplicate is not an error
				if strings.Contains(msg, "duplicate") {
					return nil
				}
				return fmt.Errorf("upload failed: %s", msg)
			}
		}
		return fmt.Errorf("upload failed: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

func isMediaFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".avif",
		".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v", ".3gp",
		".raw", ".cr2", ".nef", ".arw", ".dng", ".orf", ".rw2":
		return true
	}
	return false
}

// UploadFile uploads a single file to Immich
func (i *Importer) UploadFile(ctx context.Context, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	return i.uploadAsset(ctx, filepath.Base(filePath), content, info.ModTime())
}
