package main

import (
	"context"
	"fmt"
	"log"

	"github.com/immich-app/immich-importer/internal/config"
	"github.com/immich-app/immich-importer/internal/downloader"
	"github.com/immich-app/immich-importer/internal/google"
	"github.com/immich-app/immich-importer/internal/importer"
	"github.com/immich-app/immich-importer/internal/state"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds the application state
type App struct {
	ctx        context.Context
	serverURL  string
	setupToken string
	config     *config.Config
	state      *state.JobState
	google     *google.Client
}

// NewApp creates a new App instance
func NewApp(serverURL, setupToken string) *App {
	return &App{
		serverURL:  serverURL,
		setupToken: setupToken,
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Try to load existing config
	cfg, err := config.Load()
	if err == nil && cfg != nil {
		a.config = cfg
		log.Printf("Loaded existing config for server: %s", cfg.ServerURL)
	}

	// Try to load existing state
	st, err := state.Load()
	if err == nil && st != nil {
		a.state = st
		log.Printf("Loaded existing state: %s", st.Status)
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Save state on shutdown
	if a.state != nil {
		a.state.Save()
	}
}

// GetInitialState returns the initial app state for the frontend
func (a *App) GetInitialState() map[string]interface{} {
	result := map[string]interface{}{
		"hasConfig":      a.config != nil,
		"hasGoogleAuth":  false,
		"serverURL":      "",
		"status":         "idle",
		"needsSetup":     a.config == nil,
		"hasSetupToken":  a.setupToken != "",
		"initialServerURL": a.serverURL,
	}

	if a.config != nil {
		result["serverURL"] = a.config.ServerURL
		result["hasGoogleAuth"] = a.config.HasGoogleTokens()
	}

	if a.state != nil {
		result["status"] = a.state.Status
	}

	return result
}

// Setup fetches config from server using the setup token
func (a *App) Setup(serverURL, token string) error {
	if serverURL == "" {
		serverURL = a.serverURL
	}
	if token == "" {
		token = a.setupToken
	}

	if serverURL == "" || token == "" {
		return fmt.Errorf("server URL and token are required")
	}

	cfg, err := config.FetchFromServer(serverURL, token)
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	a.config = cfg
	log.Printf("Setup complete for server: %s", cfg.ServerURL)

	return nil
}

// GetGoogleAuthURL initiates Google OAuth and returns the auth URL
func (a *App) GetGoogleAuthURL() (string, error) {
	if a.config == nil {
		return "", fmt.Errorf("not configured - run setup first")
	}

	client, authURL, err := google.StartOAuth(a.config.OAuth)
	if err != nil {
		return "", err
	}

	a.google = client
	return authURL, nil
}

// CompleteGoogleAuth exchanges the auth code for tokens
func (a *App) CompleteGoogleAuth(code string) error {
	if a.google == nil {
		return fmt.Errorf("OAuth not started")
	}

	if err := a.google.ExchangeCode(code); err != nil {
		return err
	}

	// Store tokens in config
	tokens := a.google.GetTokens()
	a.config.GoogleAccessToken = tokens.AccessToken
	a.config.GoogleRefreshToken = tokens.RefreshToken
	a.config.GoogleTokenExpiry = tokens.Expiry

	if err := a.config.Save(); err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	return nil
}

// ListTakeoutFiles lists Google Takeout files in Drive
func (a *App) ListTakeoutFiles() ([]google.DriveFile, error) {
	if a.config == nil || !a.config.HasGoogleTokens() {
		return nil, fmt.Errorf("Google not connected")
	}

	client, err := google.NewClientFromTokens(a.config.OAuth, a.config.GoogleAccessToken, a.config.GoogleRefreshToken, a.config.GoogleTokenExpiry)
	if err != nil {
		return nil, err
	}

	return client.ListTakeoutFiles()
}

// StartImport begins the import process
func (a *App) StartImport(fileIDs []string) error {
	if a.config == nil {
		return fmt.Errorf("not configured")
	}

	// Initialize or load state
	if a.state == nil {
		a.state = state.New()
	}

	a.state.ServerURL = a.config.ServerURL
	a.state.Status = "downloading"

	// Initialize file states
	for _, id := range fileIDs {
		a.state.AddFile(id, "", 0)
	}
	a.state.Save()

	// Start import in background
	go a.runImport()

	return nil
}

func (a *App) runImport() {
	ctx := a.ctx

	// Progress callback
	emitProgress := func(phase string, current, total int, currentFile string) {
		runtime.EventsEmit(ctx, "progress", map[string]interface{}{
			"phase":       phase,
			"current":     current,
			"total":       total,
			"currentFile": currentFile,
		})
	}

	// Create Google client
	googleClient, err := google.NewClientFromTokens(
		a.config.OAuth,
		a.config.GoogleAccessToken,
		a.config.GoogleRefreshToken,
		a.config.GoogleTokenExpiry,
	)
	if err != nil {
		runtime.EventsEmit(ctx, "error", err.Error())
		return
	}

	// Phase 1: Download files from Google Drive
	dl := downloader.New(googleClient)
	for i := range a.state.Files {
		file := &a.state.Files[i]
		if file.Downloaded {
			continue
		}

		emitProgress("downloading", i, len(a.state.Files), file.Name)

		if err := dl.DownloadFile(ctx, file); err != nil {
			a.state.LastError = err.Error()
			a.state.Save()
			runtime.EventsEmit(ctx, "error", err.Error())
			return
		}

		file.Downloaded = true
		a.state.Save()
	}

	// Phase 2: Upload to Immich
	a.state.Status = "uploading"
	a.state.Save()

	imp := importer.New(a.config.ServerURL, a.config.APIKey)
	if err := imp.ImportFiles(ctx, a.state, emitProgress); err != nil {
		a.state.LastError = err.Error()
		a.state.Save()
		runtime.EventsEmit(ctx, "error", err.Error())
		return
	}

	a.state.Status = "complete"
	a.state.Save()
	runtime.EventsEmit(ctx, "complete", nil)
}

// GetJobState returns the current job state
func (a *App) GetJobState() *state.JobState {
	return a.state
}

// CancelImport cancels the current import
func (a *App) CancelImport() error {
	if a.state != nil {
		a.state.Status = "cancelled"
		a.state.Save()
	}
	return nil
}

// ResetState clears all saved state
func (a *App) ResetState() error {
	if err := state.Clear(); err != nil {
		return err
	}
	a.state = nil
	return nil
}
