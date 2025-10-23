package fetch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHTTPCache_Miss(t *testing.T) {
	cache, err := NewHTTPCache(t.TempDir(), 1*time.Hour)
	if err != nil {
		t.Fatalf("NewHTTPCache failed: %v", err)
	}

	entry, err := cache.Get("https://example.com/test")
	if err != nil {
		t.Errorf("Get should not error on miss: %v", err)
	}
	if entry != nil {
		t.Errorf("Expected cache miss, got entry: %+v", entry)
	}
}

func TestHTTPCache_HitAndExpiration(t *testing.T) {
	cache, err := NewHTTPCache(t.TempDir(), 100*time.Millisecond) // Short TTL
	if err != nil {
		t.Fatalf("NewHTTPCache failed: %v", err)
	}

	// Store entry
	url := "https://example.com/test"
	testEntry := CacheEntry{
		URL:          url,
		Body:         []byte("test body"),
		LastModified: "Wed, 21 Oct 2015 07:28:00 GMT",
		StatusCode:   200,
	}

	if err := cache.Set(testEntry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should hit cache immediately
	entry, err := cache.Get(url)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if entry == nil {
		t.Fatal("Expected cache hit, got miss")
	}

	if string(entry.Body) != "test body" {
		t.Errorf("Body = %q, want %q", string(entry.Body), "test body")
	}
	if entry.LastModified != "Wed, 21 Oct 2015 07:28:00 GMT" {
		t.Errorf("LastModified = %q", entry.LastModified)
	}
	if entry.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", entry.StatusCode)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should miss after expiration
	entry, err = cache.Get(url)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if entry != nil {
		t.Errorf("Expected cache miss after expiration, got entry")
	}
}

func TestHTTPCache_AtomicWrite(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewHTTPCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewHTTPCache failed: %v", err)
	}

	url := "https://example.com/test"
	testEntry := CacheEntry{
		URL:        url,
		Body:       []byte("atomic write test"),
		StatusCode: 200,
	}

	if err := cache.Set(testEntry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify cache file exists and temp file is cleaned up
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	var hasCache, hasTemp bool
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			hasCache = true
		}
		if filepath.Ext(entry.Name()) == ".tmp" {
			hasTemp = true
		}
	}

	if !hasCache {
		t.Error("Expected cache file to exist")
	}
	if hasTemp {
		t.Error("Expected temp file to be cleaned up")
	}
}

func TestHTTPCache_MultipleURLs(t *testing.T) {
	cache, err := NewHTTPCache(t.TempDir(), 1*time.Hour)
	if err != nil {
		t.Fatalf("NewHTTPCache failed: %v", err)
	}

	// Store multiple entries
	urls := []string{
		"https://example.com/api/v1",
		"https://example.com/api/v2",
		"https://other.com/data",
	}

	for i, url := range urls {
		entry := CacheEntry{
			URL:        url,
			Body:       []byte(url), // Use URL as body for easy verification
			StatusCode: 200 + i,
		}
		if err := cache.Set(entry); err != nil {
			t.Fatalf("Set(%q) failed: %v", url, err)
		}
	}

	// Verify all entries independently
	for i, url := range urls {
		entry, err := cache.Get(url)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", url, err)
		}
		if entry == nil {
			t.Fatalf("Get(%q) = nil, want entry", url)
		}
		if string(entry.Body) != url {
			t.Errorf("Get(%q).Body = %q, want %q", url, string(entry.Body), url)
		}
		if entry.StatusCode != 200+i {
			t.Errorf("Get(%q).StatusCode = %d, want %d", url, entry.StatusCode, 200+i)
		}
	}
}

func TestHTTPCache_ETag(t *testing.T) {
	cache, err := NewHTTPCache(t.TempDir(), 1*time.Hour)
	if err != nil {
		t.Fatalf("NewHTTPCache failed: %v", err)
	}

	url := "https://example.com/test"
	testEntry := CacheEntry{
		URL:        url,
		Body:       []byte("test"),
		ETag:       `"abc123"`,
		StatusCode: 200,
	}

	if err := cache.Set(testEntry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	entry, err := cache.Get(url)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if entry == nil {
		t.Fatal("Expected cache hit")
	}

	if entry.ETag != `"abc123"` {
		t.Errorf("ETag = %q, want %q", entry.ETag, `"abc123"`)
	}
}
