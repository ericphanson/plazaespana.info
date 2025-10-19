package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient(5 * time.Second)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.httpClient == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}

func TestClient_FetchWithUserAgent(t *testing.T) {
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"@graph":[]}`))
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)
	_, err := client.FetchJSON(server.URL)
	if err != nil {
		t.Fatalf("FetchJSON failed: %v", err)
	}

	if capturedUserAgent == "" {
		t.Error("User-Agent header not set")
	}
	if capturedUserAgent != "madrid-events-site-generator/1.0 (https://github.com/yourusername/madrid-events)" {
		t.Errorf("Unexpected User-Agent: %s", capturedUserAgent)
	}
}
