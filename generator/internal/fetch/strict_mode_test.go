package fetch

import (
	"os"
	"testing"
	"time"
)

func TestStrictMode_BlocksExternalAPIs(t *testing.T) {
	// Set strict mode
	os.Setenv("PLAZAESPANA_NO_API", "1")
	defer os.Unsetenv("PLAZAESPANA_NO_API")

	config := DefaultDevelopmentConfig()
	client, err := NewClient(30*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Try to fetch from real Madrid API
	result := client.FetchJSON("https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json", time.UTC)

	// Should have failed with a blocking error
	if len(result.Errors) == 0 {
		t.Fatalf("Expected request to be blocked, but it succeeded")
	}

	errMsg := result.Errors[0].Error.Error()
	if !contains(errMsg, "BLOCKED") || !contains(errMsg, "PLAZAESPANA_NO_API") {
		t.Errorf("Expected blocking error message, got: %s", errMsg)
	}

	t.Logf("✅ External API request blocked as expected: %s", errMsg)
}

func TestStrictMode_AllowsFileURLs(t *testing.T) {
	// Set strict mode
	os.Setenv("PLAZAESPANA_NO_API", "1")
	defer os.Unsetenv("PLAZAESPANA_NO_API")

	config := DefaultDevelopmentConfig()
	client, err := NewClient(30*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// file:// URLs should still work
	tmpFile := t.TempDir() + "/test.json"
	os.WriteFile(tmpFile, []byte(`{"@graph":[]}`), 0644)

	result := client.FetchJSON("file://"+tmpFile, time.UTC)

	// Should succeed (file:// URLs are not blocked)
	if len(result.Errors) > 0 {
		t.Fatalf("Expected file:// URL to work, got error: %v", result.Errors[0].Error)
	}

	t.Logf("✅ file:// URL allowed as expected in strict mode")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
