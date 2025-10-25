package fetch

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CacheEntry stores cached HTTP response data.
type CacheEntry struct {
	URL          string    `json:"url"`
	Body         []byte    `json:"body"`
	LastModified string    `json:"last_modified"`
	ETag         string    `json:"etag"`
	FetchedAt    time.Time `json:"fetched_at"`
	StatusCode   int       `json:"status_code"`
}

// HTTPCache manages persistent HTTP response caching.
type HTTPCache struct {
	cacheDir     string
	ttl          time.Duration
	ttlOverrides map[string]time.Duration // URL pattern -> TTL overrides
}

// NewHTTPCache creates a cache with the given directory and TTL.
func NewHTTPCache(cacheDir string, ttl time.Duration) (*HTTPCache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache dir: %w", err)
	}
	return &HTTPCache{
		cacheDir:     cacheDir,
		ttl:          ttl,
		ttlOverrides: make(map[string]time.Duration),
	}, nil
}

// SetTTLOverride sets a custom TTL for URLs containing the given pattern.
// For example, SetTTLOverride("opendata.aemet.es", 6*time.Hour) makes AEMET requests
// use a 6-hour cache TTL instead of the default.
func (c *HTTPCache) SetTTLOverride(urlPattern string, ttl time.Duration) {
	c.ttlOverrides[urlPattern] = ttl
}

// Get retrieves cached entry if valid (not expired).
// Returns nil if cache miss or expired.
func (c *HTTPCache) Get(url string) (*CacheEntry, error) {
	path := c.cachePath(url)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing cache entry: %w", err)
	}

	// Determine TTL: check for URL pattern overrides first
	ttl := c.ttl
	for pattern, overrideTTL := range c.ttlOverrides {
		if strings.Contains(url, pattern) {
			ttl = overrideTTL
			break
		}
	}

	// Check if expired
	if time.Since(entry.FetchedAt) > ttl {
		return nil, nil // Cache expired
	}

	return &entry, nil
}

// Set stores response in cache with atomic write.
func (c *HTTPCache) Set(entry CacheEntry) error {
	entry.FetchedAt = time.Now()

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache entry: %w", err)
	}

	path := c.cachePath(entry.URL)

	// Atomic write: temp file + rename
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("renaming cache: %w", err)
	}

	return nil
}

// Delete removes a cache entry for the given URL.
// Returns nil if the entry doesn't exist (idempotent).
func (c *HTTPCache) Delete(url string) error {
	path := c.cachePath(url)
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil // Already deleted, not an error
	}
	return err
}

// cachePath generates a safe filename from URL using SHA256 hash.
func (c *HTTPCache) cachePath(url string) string {
	hash := sha256.Sum256([]byte(url))
	filename := fmt.Sprintf("%x.json", hash[:8]) // First 8 bytes of hash
	return filepath.Join(c.cacheDir, filename)
}
