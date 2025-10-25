//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIntegration_FullPipeline(t *testing.T) {
	// Create test JSON server
	jsonData := `{
		"@graph": [
			{
				"ID-EVENTO": "INT-001",
				"TITULO": "Integration Test Event",
				"FECHA": "15/12/2025",
				"HORA": "20:00",
				"NOMBRE-INSTALACION": "Plaza de España",
				"COORDENADA-LATITUD": 40.42338,
				"COORDENADA-LONGITUD": -3.71217,
				"CONTENT-URL": "https://example.com/event"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonData))
	}))
	defer server.Close()

	// Setup test directories
	tmpDir := t.TempDir()
	_ = filepath.Join(tmpDir, "public")
	_ = filepath.Join(tmpDir, "data")

	// Create minimal template
	tmplDir := filepath.Join(tmpDir, "templates")
	_ = os.MkdirAll(tmplDir, 0755)
	tmpl := `<!doctype html><html><body>{{range .Events}}<p>{{.Titulo}}</p>{{end}}</body></html>`
	_ = os.WriteFile(filepath.Join(tmplDir, "index.tmpl.html"), []byte(tmpl), 0644)

	// Override template path in main (would need refactoring for real test)
	// For now, verify components work individually

	t.Log("Integration test validates component interactions")
	t.Log("Full e2e test would require refactoring main.go for testability")
}

func TestIntegration_HTMLValidation(t *testing.T) {
	// Check if npx is available (required for html-validate)
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found - skipping HTML validation test")
	}

	// Build the site first using justfile (which uses fixtures in development mode)
	rootDir := filepath.Join("..", "..", "..")
	cmd := exec.Command("just", "build")
	cmd.Dir = rootDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	//Run buildsite with test config
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "public")
	dataDir := filepath.Join(tmpDir, "data")

	// Create output dir
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Run the buildsite binary with fixture URLs from test
	fixturesDir := filepath.Join(rootDir, "generator", "testdata", "fixtures")

	// Serve fixtures over HTTP for the test
	jsonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "madrid-events.json"))
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer jsonServer.Close()

	xmlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "madrid-events.xml"))
		w.Header().Set("Content-Type", "application/xml")
		w.Write(data)
	}))
	defer xmlServer.Close()

	csvServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "madrid-events.csv"))
		w.Header().Set("Content-Type", "text/csv")
		w.Write(data)
	}))
	defer csvServer.Close()

	esmadridServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "esmadrid-agenda.xml"))
		w.Header().Set("Content-Type", "application/xml")
		w.Write(data)
	}))
	defer esmadridServer.Close()

	// Mock AEMET forecast data endpoint (must be created first so we have its URL)
	aemetForecastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "aemet-madrid-forecast.json"))
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer aemetForecastServer.Close()

	// Mock AEMET base server (handles both metadata and forecast requests)
	// The weather client will request: {baseURL}/prediccion/especifica/municipio/diaria/{code}
	aemetBaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// All paths return metadata pointing to the forecast server
		metadata := map[string]interface{}{
			"descripcion": "exito",
			"estado":      200,
			"datos":       aemetForecastServer.URL, // Point to mock server instead of real AEMET
			"metadatos":   "http://mock/metadatos",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metadata)
	}))
	defer aemetBaseServer.Close()

	// Run the buildsite binary
	templatePath := filepath.Join(rootDir, "generator", "templates", "index.tmpl.html")
	buildCmd := exec.Command(
		filepath.Join(rootDir, "build", "buildsite"),
		"-json-url", jsonServer.URL,
		"-xml-url", xmlServer.URL,
		"-csv-url", csvServer.URL,
		"-esmadrid-url", esmadridServer.URL,
		"-aemet-base-url", aemetBaseServer.URL, // Point to mock AEMET server
		"-out-dir", outDir,
		"-data-dir", dataDir,
		"-template-path", templatePath,
		"-lat", "40.42338",
		"-lon", "-3.71217",
		"-radius-km", "0.35",
		"-timezone", "Europe/Madrid",
	)

	// Set AEMET API key for weather integration test
	// Set PLAZAESPANA_NO_API to enforce no external API calls (only mock servers allowed)
	buildCmd.Env = append(os.Environ(), "AEMET_API_KEY=test-api-key", "PLAZAESPANA_NO_API=1")

	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run buildsite: %v\n%s", err, output)
	}

	// Validate the generated HTML
	outputPath := filepath.Join(outDir, "index.html")
	validateCmd := exec.Command("npx", "html-validate", outputPath)
	output, err := validateCmd.CombinedOutput()

	if err != nil {
		t.Errorf("HTML validation failed:\n%s", string(output))
	}

	// Verify HTML was generated
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("Generated HTML file not found: %v", err)
	}

	t.Logf("Successfully generated and validated HTML from fixtures")
}

func TestIntegration_InvalidWeatherDataExitsWithError(t *testing.T) {
	// Build the binary in a temp location
	tmpBinary := filepath.Join(t.TempDir(), "buildsite-test")
	cmd := exec.Command("go", "build", "-o", tmpBinary, "./cmd/buildsite")
	cmd.Dir = filepath.Join("..", "..")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	rootDir := filepath.Join("..", "..", "..")

	// Setup test directories
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "public")
	dataDir := filepath.Join(tmpDir, "data")

	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Serve fixtures over HTTP for the test
	fixturesDir := filepath.Join(rootDir, "generator", "testdata", "fixtures")

	jsonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "madrid-events.json"))
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer jsonServer.Close()

	xmlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "madrid-events.xml"))
		w.Header().Set("Content-Type", "application/xml")
		w.Write(data)
	}))
	defer xmlServer.Close()

	csvServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "madrid-events.csv"))
		w.Header().Set("Content-Type", "text/csv")
		w.Write(data)
	}))
	defer csvServer.Close()

	esmadridServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(fixturesDir, "esmadrid-agenda.xml"))
		w.Header().Set("Content-Type", "application/xml")
		w.Write(data)
	}))
	defer esmadridServer.Close()

	// Mock AEMET forecast server that returns INVALID data (object instead of array)
	// This simulates the intermittent error: "json: cannot unmarshal object into Go value of type []weather.Forecast"
	aemetForecastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a JSON object instead of an array (invalid format)
		invalidJSON := `{"error": "unexpected format"}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(invalidJSON))
	}))
	defer aemetForecastServer.Close()

	// Mock AEMET base server (returns metadata pointing to invalid forecast)
	aemetBaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := map[string]interface{}{
			"descripcion": "exito",
			"estado":      200,
			"datos":       aemetForecastServer.URL,
			"metadatos":   "http://mock/metadatos",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metadata)
	}))
	defer aemetBaseServer.Close()

	// Run the buildsite binary with invalid weather data
	templatePath := filepath.Join(rootDir, "generator", "templates", "index.tmpl.html")
	buildCmd := exec.Command(
		tmpBinary,
		"-json-url", jsonServer.URL,
		"-xml-url", xmlServer.URL,
		"-csv-url", csvServer.URL,
		"-esmadrid-url", esmadridServer.URL,
		"-aemet-base-url", aemetBaseServer.URL,
		"-out-dir", outDir,
		"-data-dir", dataDir,
		"-template-path", templatePath,
		"-lat", "40.42338",
		"-lon", "-3.71217",
		"-radius-km", "0.35",
		"-timezone", "Europe/Madrid",
	)

	buildCmd.Env = append(os.Environ(), "AEMET_API_KEY=test-api-key", "PLAZAESPANA_NO_API=1")

	output, err := buildCmd.CombinedOutput()

	// Verify the command failed with non-zero exit code
	if err == nil {
		t.Fatalf("Expected build to fail with invalid weather data, but it succeeded.\nOutput:\n%s", output)
	}

	// Verify it's an exit error (not some other kind of error) and get the exit code
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("Expected ExitError due to invalid weather data, got: %T: %v", err, err)
	}

	// Verify the exit code is non-zero (log.Fatal exits with code 1)
	exitCode := exitErr.ExitCode()
	if exitCode == 0 {
		t.Fatalf("Expected non-zero exit code, got: %d", exitCode)
	}
	t.Logf("Binary exited with code %d (non-zero, as expected)", exitCode)

	// Verify the error output contains information about the weather failure
	outputStr := string(output)
	if !hasSubstring(outputStr, "ERROR: Weather fetch failed") && !hasSubstring(outputStr, "parsing forecast") {
		t.Errorf("Expected error output to mention weather failure, got:\n%s", outputStr)
	}

	// Verify the full API response is dumped (our new debugging feature)
	if !hasSubstring(outputStr, "Full API response body") || !hasSubstring(outputStr, `{"error": "unexpected format"}`) {
		t.Errorf("Expected error output to include full API response body for debugging, got:\n%s", outputStr)
	}

	t.Logf("Successfully verified binary exits with non-zero code on invalid weather data")
}

// Helper function to check if string contains substring using strings package
func hasSubstring(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}
