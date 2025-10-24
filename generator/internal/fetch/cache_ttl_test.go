package fetch

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestHTTPCache_TTLOverride(t *testing.T) {
	// Create cache with 1-hour default TTL
	cacheDir := t.TempDir()
	cache, err := NewHTTPCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Set 6-hour TTL override for AEMET URLs
	cache.SetTTLOverride("opendata.aemet.es", 6*time.Hour)

	// Create two cache entries with timestamps 2 hours ago
	twoHoursAgo := time.Now().Add(-2 * time.Hour)

	madridEntry := CacheEntry{
		URL:       "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
		Body:      []byte(`{"events":[]}`),
		FetchedAt: twoHoursAgo,
		StatusCode: 200,
	}

	aemetEntry := CacheEntry{
		URL:       "https://opendata.aemet.es/opendata/api/prediccion/especifica/municipio/diaria/28079",
		Body:      []byte(`{"forecast":[]}`),
		FetchedAt: twoHoursAgo,
		StatusCode: 200,
	}

	// Manually write cache entries with old timestamps
	for _, entry := range []CacheEntry{madridEntry, aemetEntry} {
		data, _ := json.MarshalIndent(entry, "", "  ")
		path := cache.cachePath(entry.URL)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("Failed to write cache entry: %v", err)
		}
	}

	// Madrid entry should be expired (2 hours > 1 hour default TTL)
	madridCached, err := cache.Get(madridEntry.URL)
	if err != nil {
		t.Fatalf("Error getting Madrid entry: %v", err)
	}
	if madridCached != nil {
		t.Errorf("Expected Madrid entry to be expired (2h > 1h TTL), but it was cached")
	}

	// AEMET entry should still be valid (2 hours < 6 hour override TTL)
	aemetCached, err := cache.Get(aemetEntry.URL)
	if err != nil {
		t.Fatalf("Error getting AEMET entry: %v", err)
	}
	if aemetCached == nil {
		t.Errorf("Expected AEMET entry to be cached (2h < 6h TTL), but it was expired")
	} else {
		t.Logf("✅ AEMET entry cached as expected (2h < 6h TTL)")
	}

	t.Logf("✅ TTL override working correctly")
}
