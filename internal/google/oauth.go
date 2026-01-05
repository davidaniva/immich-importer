package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/davidaniva/immich-importer/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Client handles Google API interactions
type Client struct {
	oauth2Config *oauth2.Config
	token        *oauth2.Token
	httpClient   *http.Client
	callbackChan chan string
	listener     net.Listener
}

// Tokens holds OAuth token information
type Tokens struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// DriveFile represents a file in Google Drive
type DriveFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size,string"`
	MimeType string `json:"mimeType"`
}

// DriveFileList is the response from Drive API
type DriveFileList struct {
	Files         []DriveFile `json:"files"`
	NextPageToken string      `json:"nextPageToken"`
}

var scopes = []string{
	"https://www.googleapis.com/auth/drive.readonly",
}

// Fixed port for OAuth callback - must match Google Cloud Console redirect URI
const oauthCallbackPort = 8085

// StartOAuth initiates the OAuth flow and returns the auth URL
func StartOAuth(cfg config.OAuthConfig) (*Client, string, error) {
	// Use fixed port so redirect URI can be registered in Google Cloud Console
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", oauthCallbackPort))
	if err != nil {
		return nil, "", fmt.Errorf("failed to start callback server on port %d: %w", oauthCallbackPort, err)
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", oauthCallbackPort)

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	client := &Client{
		oauth2Config: oauth2Config,
		callbackChan: make(chan string, 1),
		listener:     listener,
	}

	// Start callback server
	go client.startCallbackServer()

	// Generate auth URL
	authURL := oauth2Config.AuthCodeURL("state",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	return client, authURL, nil
}

func (c *Client) startCallbackServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		// Send code to channel
		c.callbackChan <- code

		// Show success page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Success</title></head>
<body style="font-family: sans-serif; text-align: center; padding: 50px;">
  <h1>Authentication Successful</h1>
  <p>You can close this window and return to the app.</p>
  <script>window.close();</script>
</body>
</html>`)
	})

	http.Serve(c.listener, mux)
}

// WaitForCallback waits for the OAuth callback
func (c *Client) WaitForCallback(timeout time.Duration) (string, error) {
	select {
	case code := <-c.callbackChan:
		c.listener.Close()
		return code, nil
	case <-time.After(timeout):
		c.listener.Close()
		return "", fmt.Errorf("timeout waiting for OAuth callback")
	}
}

// ExchangeCode exchanges an auth code for tokens
func (c *Client) ExchangeCode(code string) error {
	ctx := context.Background()
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	c.token = token
	c.httpClient = c.oauth2Config.Client(ctx, token)
	return nil
}

// GetTokens returns the current tokens
func (c *Client) GetTokens() Tokens {
	if c.token == nil {
		return Tokens{}
	}
	return Tokens{
		AccessToken:  c.token.AccessToken,
		RefreshToken: c.token.RefreshToken,
		Expiry:       c.token.Expiry,
	}
}

// NewClientFromTokens creates a client from stored tokens
func NewClientFromTokens(cfg config.OAuthConfig, accessToken, refreshToken string, expiry time.Time) (*Client, error) {
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
	}

	ctx := context.Background()
	client := &Client{
		oauth2Config: oauth2Config,
		token:        token,
		httpClient:   oauth2Config.Client(ctx, token),
	}

	return client, nil
}

// ListTakeoutFiles lists Google Takeout files in Drive
func (c *Client) ListTakeoutFiles() ([]DriveFile, error) {
	// Search for Takeout files - they're usually named "takeout-*.zip" or in a "Takeout" folder
	query := "(name contains 'takeout' and mimeType = 'application/zip') or (name contains 'Takeout' and mimeType = 'application/vnd.google-apps.folder')"

	var allFiles []DriveFile
	pageToken := ""

	for {
		url := fmt.Sprintf(
			"https://www.googleapis.com/drive/v3/files?q=%s&fields=files(id,name,size,mimeType)&pageSize=100",
			query,
		)
		if pageToken != "" {
			url += "&pageToken=" + pageToken
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("drive API error: %d", resp.StatusCode)
		}

		var result DriveFileList
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		allFiles = append(allFiles, result.Files...)

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return allFiles, nil
}

// DownloadFile downloads a file from Google Drive
func (c *Client) DownloadFile(fileID string) (*http.Response, error) {
	url := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", fileID)
	return c.httpClient.Get(url)
}

// DownloadFileRange downloads a file with Range header for resume
func (c *Client) DownloadFileRange(fileID string, startByte int64) (*http.Response, error) {
	url := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", fileID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	return c.httpClient.Do(req)
}
