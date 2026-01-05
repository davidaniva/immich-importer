package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// JobState tracks the overall import job progress
type JobState struct {
	ID          string       `json:"id"`
	ServerURL   string       `json:"serverUrl"`
	Status      string       `json:"status"` // idle, downloading, uploading, complete, error, cancelled
	Files       []FileState  `json:"files"`
	UploadState *UploadState `json:"uploadState,omitempty"`
	LastError   string       `json:"lastError,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// FileState tracks individual file download/upload progress
type FileState struct {
	DriveID         string `json:"driveId"`
	Name            string `json:"name"`
	Size            int64  `json:"size"`
	Downloaded      bool   `json:"downloaded"`
	LocalPath       string `json:"localPath,omitempty"`
	BytesDownloaded int64  `json:"bytesDownloaded"`
}

// UploadState tracks upload progress
type UploadState struct {
	TotalPhotos    int      `json:"totalPhotos"`
	UploadedPhotos int      `json:"uploadedPhotos"`
	UploadedFiles  []string `json:"uploadedFiles"`
}

// New creates a new JobState
func New() *JobState {
	return &JobState{
		ID:        uuid.New().String(),
		Status:    "idle",
		Files:     []FileState{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Load loads state from disk
func Load() (*JobState, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state JobState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Save saves state to disk
func (s *JobState) Save() error {
	path, err := statePath()
	if err != nil {
		return err
	}

	s.UpdatedAt = time.Now()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// Clear removes the state file
func Clear() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// AddFile adds a file to track
func (s *JobState) AddFile(driveID, name string, size int64) {
	// Check if already exists
	for _, f := range s.Files {
		if f.DriveID == driveID {
			return
		}
	}

	s.Files = append(s.Files, FileState{
		DriveID:         driveID,
		Name:            name,
		Size:            size,
		Downloaded:      false,
		BytesDownloaded: 0,
	})
}

// GetDownloadProgress returns download progress (0-100)
func (s *JobState) GetDownloadProgress() float64 {
	if len(s.Files) == 0 {
		return 0
	}

	var totalBytes, downloadedBytes int64
	for _, f := range s.Files {
		totalBytes += f.Size
		if f.Downloaded {
			downloadedBytes += f.Size
		} else {
			downloadedBytes += f.BytesDownloaded
		}
	}

	if totalBytes == 0 {
		return 0
	}

	return float64(downloadedBytes) / float64(totalBytes) * 100
}

// GetUploadProgress returns upload progress (0-100)
func (s *JobState) GetUploadProgress() float64 {
	if s.UploadState == nil || s.UploadState.TotalPhotos == 0 {
		return 0
	}
	return float64(s.UploadState.UploadedPhotos) / float64(s.UploadState.TotalPhotos) * 100
}

func statePath() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func appDataDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, "Library", "Application Support", "ImmichImporter")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		baseDir = filepath.Join(appData, "ImmichImporter")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		baseDir = filepath.Join(configDir, "immich-importer")
	}

	return baseDir, nil
}
