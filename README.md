# Immich Google Photos Importer

A CLI tool for importing Google Photos Takeout exports to Immich.

## Features

- Downloads Google Takeout files directly from Google Drive
- Uploads photos and videos to your Immich server
- **Resumable** - safe to interrupt with Ctrl+C, run again to continue
- Preserves metadata (dates, albums)
- Cross-platform: Windows, macOS, Linux

## Quick Start

### Option 1: Via Immich Web UI (Recommended)

1. Go to your Immich server's **Utilities > Import from Google Photos**
2. Click **"Download Importer App"**
3. Run the downloaded file - it will download this CLI and launch it
4. Follow the prompts to connect Google and import your photos

### Option 2: Manual Setup

1. Download the latest release for your platform from [Releases](https://github.com/davidaniva/immich-importer/releases)
2. Get a setup token from your Immich server:
   ```bash
   curl -X POST https://your-immich-server/api/importer/setup-token \
     -H "Authorization: Bearer YOUR_API_KEY"
   ```
3. Run the importer:
   ```bash
   ./immich-importer --server https://your-immich-server --token YOUR_SETUP_TOKEN
   ```

## Usage

```
immich-importer [flags]

Flags:
  --server string   Immich server URL (e.g., https://photos.example.com)
  --token string    Setup token from Immich server
```

## How It Works

1. **Setup**: Fetches configuration (API key, OAuth credentials) from your Immich server
2. **Google Auth**: Opens browser for Google OAuth, you paste the code back
3. **File Selection**: Lists Takeout files in your Google Drive, you select which to import
4. **Download**: Downloads selected files from Google Drive (resumable)
5. **Upload**: Extracts and uploads photos to Immich (resumable)

## Resumability

The importer saves its state to disk after each file:

- **Downloads**: Uses HTTP Range headers to resume partial downloads
- **Uploads**: Tracks uploaded files, skips them on restart
- **Interrupt anytime**: Press Ctrl+C to pause, run again to continue

State is saved to:
- macOS: `~/Library/Application Support/ImmichImporter/`
- Windows: `%APPDATA%\ImmichImporter\`
- Linux: `~/.config/immich-importer/`

## Building

```bash
# Build for current platform
go build -o immich-importer .

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o immich-importer-darwin-arm64 .
GOOS=linux GOARCH=amd64 go build -o immich-importer-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o immich-importer-windows-amd64.exe .
```

## Security

- Setup tokens expire after 30 days
- API keys have limited permissions (upload only)
- Google OAuth tokens stored locally (consider using OS keychain for production)
- Revoke API key from Immich after import if desired
