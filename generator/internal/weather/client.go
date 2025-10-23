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
}

// NewClient creates a new weather client
func NewClient(apiKey, municipalityCode string, fetchClient *fetch.Client) *Client {
	return &Client{
		apiKey:           apiKey,
		fetchClient:      fetchClient,
		municipalityCode: municipalityCode,
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
	metadataURL := fmt.Sprintf("https://opendata.aemet.es/opendata/api/prediccion/especifica/municipio/diaria/%s", c.municipalityCode)

	metadataBody, err := c.fetchWithAPIKey(metadataURL)
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
	forecastBody, err := c.fetchWithAPIKey(metadata.DataURL)
	if err != nil {
		return nil, fmt.Errorf("fetching forecast data: %w", err)
	}

	// Parse forecast response (it's an array with single element)
	var forecasts []Forecast
	if err := json.Unmarshal(forecastBody, &forecasts); err != nil {
		return nil, fmt.Errorf("parsing forecast: %w", err)
	}

	if len(forecasts) == 0 {
		return nil, fmt.Errorf("AEMET returned empty forecast array")
	}

	return &forecasts[0], nil
}

// fetchWithAPIKey makes an HTTP request with the AEMET API key header
// Uses the fetch client's HTTPCache system for caching, throttling, and audit trail
func (c *Client) fetchWithAPIKey(url string) ([]byte, error) {
	// Use fetch client with custom header for API key
	// AEMET uses lowercase "api_key" header
	headers := map[string]string{
		"api_key": c.apiKey,
	}
	return c.fetchClient.FetchWithHeaders(url, headers)
}
