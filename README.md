# Immich Google Photos Importer

A desktop application for importing Google Photos Takeout exports to Immich.

## Features

- Downloads Google Takeout files directly from Google Drive
- Uploads photos and videos to your Immich server
- Resumable downloads and uploads - safe to close and restart
- Preserves metadata (dates, albums)
- Cross-platform: Windows, macOS, Linux

## How It Works

1. **Get Started**: Download the personalized bootstrap binary from your Immich web UI
2. **Run Bootstrap**: The bootstrap downloads this app and launches it
3. **Connect Google**: Sign in with Google to access your Takeout files
4. **Select Files**: Choose which Takeout files to import
5. **Import**: The app downloads from Google Drive and uploads to Immich
6. **Resume**: If interrupted, restart the app to continue where you left off

## Building

### Prerequisites

- Go 1.22+
- Node.js 20+
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Development

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in development mode
wails dev
```

### Build

```bash
# Build for current platform
wails build

# Cross-compile (requires platform-specific setup)
wails build -platform darwin/amd64
wails build -platform darwin/arm64
wails build -platform linux/amd64
wails build -platform windows/amd64
```

## Architecture

```
immich-importer/
├── main.go              # Wails app entry
├── app.go               # Go backend with exported methods
├── internal/
│   ├── config/          # Config storage (JSON in app data dir)
│   ├── state/           # Job state persistence for resume
│   ├── google/          # Google OAuth and Drive API
│   ├── downloader/      # Resumable file downloads
│   └── importer/        # Immich upload with checkpointing
└── frontend/
    └── src/
        ├── App.svelte
        └── lib/
            ├── Setup.svelte        # Server connection
            ├── GoogleConnect.svelte # Google OAuth
            ├── FileSelector.svelte  # Takeout file selection
            ├── Progress.svelte      # Import progress
            └── Complete.svelte      # Success screen
```

## State Persistence

All state is saved to the platform-specific app data directory:

- **macOS**: `~/Library/Application Support/ImmichImporter/`
- **Windows**: `%APPDATA%\ImmichImporter\`
- **Linux**: `~/.config/immich-importer/`

Files:
- `config.json` - Server URL, API key, Google tokens
- `state.json` - Import job progress
- `downloads/` - Downloaded Takeout files (temporary)

## Resumability

The app is designed to be resumable at any point:

1. **Download Resume**: Uses HTTP Range headers to continue partial downloads
2. **Upload Resume**: Tracks uploaded files in state.json, skips on restart
3. **Google Tokens**: Stored persistently, refreshed automatically

## Security

- API key has limited permissions (upload only)
- Google OAuth tokens stored in app data (consider keychain for production)
- All communication over HTTPS
