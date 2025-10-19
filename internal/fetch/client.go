package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps an HTTP client for fetching Madrid event data.
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a fetch client with the given timeout.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent: "madrid-events-site-generator/1.0 (https://github.com/yourusername/madrid-events)",
	}
}

// FetchJSON fetches and decodes JSON from the given URL.
func (c *Client) FetchJSON(url string) (*JSONResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result JSONResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}

	return &result, nil
}
