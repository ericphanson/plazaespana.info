package weather

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/fetch"
)

func TestFetchForecast(t *testing.T) {
	// Load fixture data
	metadataJSON, err := os.ReadFile("../../testdata/fixtures/aemet-madrid-metadata.json")
	if err != nil {
		t.Fatalf("Failed to load metadata fixture: %v", err)
	}

	forecastJSON, err := os.ReadFile("../../testdata/fixtures/aemet-madrid-forecast.json")
	if err != nil {
		t.Fatalf("Failed to load forecast fixture: %v", err)
	}

	// Parse metadata to modify it
	var metadata MetadataResponse
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		t.Fatalf("Failed to parse metadata fixture: %v", err)
	}

	// Create mock server that serves both metadata and forecast
	requestCount := 0
	var serverURL string
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Check API key header
		apiKey := r.Header.Get("api_key")
		if apiKey != "test-api-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Serve different responses based on path
		if r.URL.Path == "/prediccion/especifica/municipio/diaria/28079" {
			// Metadata request - return metadata with datos URL pointing to this server
			metadata.DataURL = serverURL + "/forecast-data"
			updatedMetadata, _ := json.Marshal(metadata)
			w.Header().Set("Content-Type", "application/json")
			w.Write(updatedMetadata)
		} else if r.URL.Path == "/forecast-data" {
			// Forecast data request
			w.Header().Set("Content-Type", "application/json")
			w.Write(forecastJSON)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer testServer.Close()
	serverURL = testServer.URL

	// Create fetch client with test cache
	tempDir := t.TempDir()
	modeConfig := fetch.ModeConfig{
		Mode:     "test",
		CacheTTL: 1 * time.Hour,
		MinDelay: 0,
	}
	fetchClient, err := fetch.NewClient(30*time.Second, modeConfig, tempDir)
	if err != nil {
		t.Fatalf("Failed to create fetch client: %v", err)
	}

	// Create weather client with test base URL
	weatherClient := NewClientWithBaseURL("test-api-key", "28079", fetchClient, testServer.URL)

	// Test 1: First fetch (should make 2 HTTP requests: metadata + datos)
	requestCount = 0
	forecast, err := weatherClient.FetchForecast()
	if err != nil {
		t.Fatalf("Failed to fetch forecast: %v", err)
	}
	if forecast == nil {
		t.Fatal("Forecast should not be nil")
	}
	if requestCount != 2 {
		t.Errorf("Expected 2 requests (metadata + datos), got %d", requestCount)
	}

	// Verify forecast structure
	if forecast.Origin.Producer == "" {
		t.Error("Producer should not be empty")
	}
	if len(forecast.Prediction.Days) == 0 {
		t.Error("Should have at least one day of forecast")
	}

	// Test 2: Second fetch (should use cache, 0 HTTP requests)
	requestCount = 0
	forecast2, err := weatherClient.FetchForecast()
	if err != nil {
		t.Fatalf("Failed to fetch cached forecast: %v", err)
	}
	if forecast2 == nil {
		t.Fatal("Cached forecast should not be nil")
	}
	if requestCount != 0 {
		t.Errorf("Expected 0 requests (cache hit), got %d", requestCount)
	}

	// Verify cached forecast has same data
	if len(forecast2.Prediction.Days) != len(forecast.Prediction.Days) {
		t.Error("Cached forecast should have same number of days")
	}
}

func TestFetchForecast_NoAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	modeConfig := fetch.ModeConfig{
		Mode:     "test",
		CacheTTL: 1 * time.Hour,
		MinDelay: 0,
	}
	fetchClient, err := fetch.NewClient(30*time.Second, modeConfig, tempDir)
	if err != nil {
		t.Fatalf("Failed to create fetch client: %v", err)
	}

	client := NewClient("", "28079", fetchClient)
	_, err = client.FetchForecast()

	if err == nil {
		t.Error("Expected error when API key is missing, got nil")
	}
	if err.Error() != "AEMET API key not provided" {
		t.Errorf("Expected 'AEMET API key not provided' error, got: %v", err)
	}
}

func TestMetadataResponseParsing(t *testing.T) {
	// Load fixture
	data, err := os.ReadFile("../../testdata/fixtures/aemet-madrid-metadata.json")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	var metadata MetadataResponse
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// Validate structure
	if metadata.State != 200 {
		t.Errorf("Expected state=200, got %d", metadata.State)
	}
	if metadata.DataURL == "" {
		t.Error("DataURL should not be empty")
	}
	if metadata.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestForecastResponseParsing(t *testing.T) {
	// Load fixture
	data, err := os.ReadFile("../../testdata/fixtures/aemet-madrid-forecast.json")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	var forecasts []Forecast
	if err := json.Unmarshal(data, &forecasts); err != nil {
		t.Fatalf("Failed to parse forecast: %v", err)
	}

	if len(forecasts) == 0 {
		t.Fatal("Forecast array should not be empty")
	}

	forecast := forecasts[0]

	// Validate basic structure
	if forecast.Origin.Producer == "" {
		t.Error("Producer should not be empty")
	}
	if len(forecast.Prediction.Days) == 0 {
		t.Error("Should have at least one day of forecast")
	}

	// Check first day structure
	day := forecast.Prediction.Days[0]
	if day.Date == "" {
		t.Error("Date should not be empty")
	}
	if day.Temperature.Max == 0 && day.Temperature.Min == 0 {
		t.Error("Temperature data missing")
	}
	if len(day.SkyState) == 0 {
		t.Error("SkyState should not be empty")
	}
}
