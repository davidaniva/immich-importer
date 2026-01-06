package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/davidaniva/immich-importer/internal/config"
	"github.com/davidaniva/immich-importer/internal/downloader"
	"github.com/davidaniva/immich-importer/internal/google"
	"github.com/davidaniva/immich-importer/internal/importer"
	"github.com/davidaniva/immich-importer/internal/state"
)

func main() {
	serverURL := flag.String("server", "", "Immich server URL")
	setupToken := flag.String("token", "", "Setup token from Immich")
	flag.Parse()

	fmt.Println("Immich Google Photos Importer")
	fmt.Println("==============================")
	fmt.Println()

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nInterrupted. Saving state...")
		cancel()
	}()

	// Try to load existing config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Note: Could not load existing config: %v\n", err)
	}

	// If no config, need to set up
	if cfg == nil {
		if *serverURL == "" || *setupToken == "" {
			fmt.Println("No existing configuration found.")
			fmt.Println("Please provide --server and --token flags, or run the bootstrap binary.")
			os.Exit(1)
		}

		fmt.Printf("Fetching configuration from %s...\n", *serverURL)
		cfg, err = config.FetchFromServer(*serverURL, *setupToken)
		if err != nil {
			fmt.Printf("Error: Failed to fetch config: %v\n", err)
			os.Exit(1)
		}

		if err := cfg.Save(); err != nil {
			fmt.Printf("Warning: Could not save config: %v\n", err)
		}
		fmt.Println("Configuration saved.")
	} else {
		fmt.Printf("Using existing configuration for: %s\n", cfg.ServerURL)
	}

	// Try to load existing state
	jobState, _ := state.Load()
	if jobState != nil && jobState.Status != "complete" && jobState.Status != "idle" {
		fmt.Printf("Found existing import job (status: %s)\n", jobState.Status)
		fmt.Print("Resume previous import? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "" && answer != "y" && answer != "yes" {
			jobState = nil
		}
	}

	// Check Google auth
	if !cfg.HasGoogleTokens() {
		fmt.Println()
		fmt.Println("Google account not connected. Starting OAuth flow...")
		if err := doGoogleAuth(cfg); err != nil {
			fmt.Printf("Error: Google authentication failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Google account connected successfully!")
	}

	// Create Google client
	googleClient, err := google.NewClientFromTokens(
		cfg.OAuth,
		cfg.GoogleAccessToken,
		cfg.GoogleRefreshToken,
		cfg.GoogleTokenExpiry,
	)
	if err != nil {
		fmt.Printf("Error: Failed to create Google client: %v\n", err)
		os.Exit(1)
	}

	// If no existing job, list files and let user select
	if jobState == nil || len(jobState.Files) == 0 {
		fmt.Println()
		fmt.Println("Searching for Google Takeout files in your Drive...")
		files, err := googleClient.ListTakeoutFiles()
		if err != nil {
			fmt.Printf("Error: Failed to list files: %v\n", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Println("No Takeout files found in your Google Drive.")
			fmt.Println("Make sure you've created a Google Takeout export and it's saved to Drive.")
			os.Exit(0)
		}

		fmt.Printf("\nFound %d Takeout file(s):\n", len(files))
		for i, f := range files {
			fmt.Printf("  [%d] %s (%.2f MB)\n", i+1, f.Name, float64(f.Size)/1024/1024)
		}

		fmt.Print("\nEnter file numbers to import (comma-separated, or 'all'): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var selectedFiles []google.DriveFile
		if strings.ToLower(input) == "all" {
			selectedFiles = files
		} else {
			for _, numStr := range strings.Split(input, ",") {
				numStr = strings.TrimSpace(numStr)
				num, err := strconv.Atoi(numStr)
				if err != nil || num < 1 || num > len(files) {
					fmt.Printf("Invalid selection: %s\n", numStr)
					continue
				}
				selectedFiles = append(selectedFiles, files[num-1])
			}
		}

		if len(selectedFiles) == 0 {
			fmt.Println("No files selected. Exiting.")
			os.Exit(0)
		}

		// Create new job state
		jobState = state.New()
		jobState.ServerURL = cfg.ServerURL
		for _, f := range selectedFiles {
			jobState.AddFile(f.ID, f.Name, f.Size)
		}
		jobState.Save()
	}

	// Start import
	fmt.Println()
	fmt.Println("Starting import...")
	fmt.Println("(Press Ctrl+C to pause - you can resume later)")
	fmt.Println()

	if err := runImport(ctx, cfg, jobState, googleClient); err != nil {
		if ctx.Err() != nil {
			fmt.Println("\nImport paused. Run again to resume.")
			os.Exit(0)
		}
		fmt.Printf("Error: Import failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Import complete!")
	fmt.Printf("Visit %s to see your photos.\n", cfg.ServerURL)
}

func doGoogleAuth(cfg *config.Config) error {
	client, authURL, err := google.StartOAuth(cfg.OAuth)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Please open this URL in your browser:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("Waiting for authorization callback...")

	// Wait for the OAuth callback (5 minute timeout)
	code, err := client.WaitForCallback(5 * time.Minute)
	if err != nil {
		return fmt.Errorf("OAuth callback failed: %w", err)
	}

	fmt.Println("Authorization received, exchanging code...")

	if err := client.ExchangeCode(code); err != nil {
		return err
	}

	tokens := client.GetTokens()
	cfg.GoogleAccessToken = tokens.AccessToken
	cfg.GoogleRefreshToken = tokens.RefreshToken
	cfg.GoogleTokenExpiry = tokens.Expiry

	return cfg.Save()
}

func runImport(ctx context.Context, cfg *config.Config, jobState *state.JobState, googleClient *google.Client) error {
	// Phase 1: Download
	jobState.Status = "downloading"
	jobState.Save()

	dl := downloader.New(googleClient)
	for i := range jobState.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		file := &jobState.Files[i]
		if file.Downloaded {
			fmt.Printf("[%d/%d] %s - already downloaded\n", i+1, len(jobState.Files), file.Name)
			continue
		}

		fmt.Printf("[%d/%d] Downloading %s...\n", i+1, len(jobState.Files), file.Name)
		if err := dl.DownloadFile(ctx, file); err != nil {
			jobState.Save()
			return fmt.Errorf("download failed: %w", err)
		}
		file.Downloaded = true
		jobState.Save()
		fmt.Printf("         Downloaded to %s\n", file.LocalPath)
	}

	// Phase 2: Upload
	jobState.Status = "uploading"
	jobState.Save()

	fmt.Println()
	fmt.Println("Uploading to Immich...")

	imp := importer.New(cfg.ServerURL, cfg.APIKey)
	progress := func(phase string, current, total int, currentFile string) {
		if currentFile != "" {
			fmt.Printf("\r[%d/%d] %s", current, total, truncate(currentFile, 50))
		}
	}

	if err := imp.ImportFiles(ctx, jobState, progress); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Println()
	jobState.Status = "complete"
	jobState.Save()

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
