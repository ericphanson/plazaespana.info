package weather

import (
	"encoding/json"
	"fmt"

	"github.com/ericphanson/plazaespana.info/internal/fetch"
)

// Client handles AEMET weather API requests
type Client struct {
	apiKey           string
	fetchClient      *fetch.Client
	municipalityCode string
	baseURL          string // Base URL for AEMET API (defaults to production, overridable for tests)
}

// NewClient creates a new weather client
func NewClient(apiKey, municipalityCode string, fetchClient *fetch.Client) *Client {
	return &Client{
		apiKey:           apiKey,
		fetchClient:      fetchClient,
		municipalityCode: municipalityCode,
		baseURL:          "https://opendata.aemet.es/opendata/api",
	}
}

// NewClientWithBaseURL creates a new weather client with custom base URL (for testing)
func NewClientWithBaseURL(apiKey, municipalityCode string, fetchClient *fetch.Client, baseURL string) *Client {
	return &Client{
		apiKey:           apiKey,
		fetchClient:      fetchClient,
		municipalityCode: municipalityCode,
		baseURL:          baseURL,
	}
}

// FetchForecast fetches the weather forecast from AEMET API using the two-step process
// Step 1: Request metadata endpoint with API key
// Step 2: Fetch actual forecast data from the datos URL
func (c *Client) FetchForecast() (*Forecast, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("AEMET API key not provided")
	}

	// Step 1: Fetch metadata to get the datos URL
	// IMPORTANT: Skip cache for metadata because the datos URL expires
	metadataURL := fmt.Sprintf("%s/prediccion/especifica/municipio/diaria/%s", c.baseURL, c.municipalityCode)

	metadataBody, err := c.fetchWithAPIKey(metadataURL, true) // skipCache=true
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}

	// Parse metadata response
	var metadata MetadataResponse
	if err := json.Unmarshal(metadataBody, &metadata); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	if metadata.State != 200 {
		return nil, fmt.Errorf("AEMET API returned state %d: %s", metadata.State, metadata.Description)
	}

	if metadata.DataURL == "" {
		return nil, fmt.Errorf("AEMET metadata response missing datos URL")
	}

	// Step 2: Fetch actual forecast data using the datos URL
	// The datos URL doesn't require authentication
	// We CAN cache this response since it's the actual forecast data
	forecastBody, err := c.fetchWithAPIKey(metadata.DataURL, false) // skipCache=false, allow caching
	if err != nil {
		return nil, fmt.Errorf("fetching forecast data: %w", err)
	}

	// Parse forecast response (it's an array with single element)
	var forecasts []Forecast
	if err := json.Unmarshal(forecastBody, &forecasts); err != nil {
		// Dump the full response for debugging
		return nil, fmt.Errorf("parsing forecast: %w\nFull API response body:\n%s", err, string(forecastBody))
	}

	if len(forecasts) == 0 {
		return nil, fmt.Errorf("AEMET returned empty forecast array")
	}

	return &forecasts[0], nil
}

// fetchWithAPIKey makes an HTTP request with the AEMET API key header
// Uses the fetch client's HTTPCache system for caching, throttling, and audit trail
// If skipCache is true, bypasses the cache for this request (but still caches the response for future use)
func (c *Client) fetchWithAPIKey(url string, skipCache bool) ([]byte, error) {
	// Use fetch client with custom header for API key
	// AEMET uses lowercase "api_key" header
	headers := map[string]string{
		"api_key": c.apiKey,
	}
	return c.fetchClient.FetchWithHeaders(url, headers, skipCache)
}
