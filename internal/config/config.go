package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Config holds the application configuration
type Config struct {
	ServerURL          string    `json:"serverUrl"`
	APIKey             string    `json:"apiKey"`
	OAuth              OAuthConfig `json:"oauth"`
	GoogleAccessToken  string    `json:"googleAccessToken,omitempty"`
	GoogleRefreshToken string    `json:"googleRefreshToken,omitempty"`
	GoogleTokenExpiry  time.Time `json:"googleTokenExpiry,omitempty"`
}

// OAuthConfig holds Google OAuth credentials
type OAuthConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// ServerConfigResponse is the response from /api/importer/config/:token
type ServerConfigResponse struct {
	ServerURL string `json:"serverUrl"`
	APIKey    string `json:"apiKey"`
	OAuth     struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	} `json:"oauth"`
}

// HasGoogleTokens returns true if Google tokens are stored
func (c *Config) HasGoogleTokens() bool {
	return c.GoogleAccessToken != "" && c.GoogleRefreshToken != ""
}

// FetchFromServer fetches config from the Immich server
func FetchFromServer(serverURL, token string) (*Config, error) {
	url := fmt.Sprintf("%s/api/importer/config/%s", serverURL, token)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var serverResp ServerConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&serverResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &Config{
		ServerURL: serverResp.ServerURL,
		APIKey:    serverResp.APIKey,
		OAuth: OAuthConfig{
			ClientID:     serverResp.OAuth.ClientID,
			ClientSecret: serverResp.OAuth.ClientSecret,
		},
	}, nil
}

// Load loads config from disk
func Load() (*Config, error) {
	path, err := configPath()
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

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves config to disk
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func configPath() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
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
	default: // Linux and others
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

// GetDownloadDir returns the directory for downloaded files
func GetDownloadDir() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	downloadDir := filepath.Join(dir, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", err
	}
	return downloadDir, nil
}
