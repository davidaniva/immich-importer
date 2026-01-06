package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davidaniva/immich-importer/internal/config"
	"github.com/davidaniva/immich-importer/internal/google"
	"github.com/davidaniva/immich-importer/internal/state"
)

// Downloader handles resumable file downloads from Google Drive
type Downloader struct {
	google *google.Client
}

// New creates a new Downloader
func New(googleClient *google.Client) *Downloader {
	return &Downloader{google: googleClient}
}

// DownloadFile downloads a file with resume support
func (d *Downloader) DownloadFile(ctx context.Context, file *state.FileState) error {
	downloadDir, err := config.GetDownloadDir()
	if err != nil {
		return err
	}

	localPath := filepath.Join(downloadDir, file.Name)
	file.LocalPath = localPath

	// Check if file already exists and get size
	var startByte int64 = 0
	if info, err := os.Stat(localPath); err == nil {
		startByte = info.Size()
		file.BytesDownloaded = startByte

		// If file is complete, skip download
		if file.Size > 0 && startByte >= file.Size {
			file.Downloaded = true
			return nil
		}
	}

	// Download with range header for resume
	resp, err := d.google.DownloadFileRange(ctx, file.DriveID, startByte)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Open file for writing (append if resuming)
	flags := os.O_CREATE | os.O_WRONLY
	if startByte > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(localPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Copy with progress tracking
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			file.BytesDownloaded += int64(n)
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read: %w", readErr)
		}
	}

	file.Downloaded = true
	return nil
}

// DownloadProgress represents download progress
type DownloadProgress struct {
	FileIndex       int
	TotalFiles      int
	BytesDownloaded int64
	TotalBytes      int64
	CurrentFile     string
}
