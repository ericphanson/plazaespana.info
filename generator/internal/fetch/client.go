package fetch

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/event"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Client wraps an HTTP client for fetching Madrid event data.
type Client struct {
	httpClient *http.Client
	userAgent  string
	cache      *HTTPCache
	throttle   *RequestThrottle
	auditor    *RequestAuditor
	config     ModeConfig
}

// NewClient creates a fetch client with the given timeout, mode config, and cache directory.
func NewClient(timeout time.Duration, config ModeConfig, cacheDir string) (*Client, error) {
	cache, err := NewHTTPCache(cacheDir, config.CacheTTL)
	if err != nil {
		return nil, fmt.Errorf("creating cache: %w", err)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent: "plazaespana-info-site-generator/1.0 (https://github.com/ericphanson/plazaespana.info)",
		cache:     cache,
		throttle:  NewRequestThrottle(config.MinDelay),
		auditor:   NewRequestAuditor(),
		config:    config,
	}, nil
}

// Auditor returns the request auditor for exporting request logs.
func (c *Client) Auditor() *RequestAuditor {
	return c.auditor
}

// Config returns the client's mode configuration.
func (c *Client) Config() ModeConfig {
	return c.config
}

// SetCacheTTLOverride sets a custom cache TTL for URLs containing the given pattern.
// For example, SetCacheTTLOverride("opendata.aemet.es", 6*time.Hour) makes AEMET
// weather requests cache for 6 hours instead of the default TTL.
func (c *Client) SetCacheTTLOverride(urlPattern string, ttl time.Duration) {
	c.cache.SetTTLOverride(urlPattern, ttl)
}

// CacheForecast manually writes data to the cache under a synthetic URL.
// This is used by the weather client to cache forecast data independently of
// the temporary AEMET URLs that expire.
func (c *Client) CacheForecast(syntheticURL string, body []byte) {
	entry := CacheEntry{
		URL:        syntheticURL,
		Body:       body,
		StatusCode: 200,
	}
	// Ignore errors - cache write failures shouldn't break the build
	_ = c.cache.Set(entry)
}

// FetchWithHeaders fetches a URL with custom HTTP headers.
// Uses the same caching, throttling, and audit trail as other fetch methods.
// Useful for APIs requiring authentication (e.g., AEMET API key header).
// If skipCache is true, bypasses the cache for this request (but still caches the response for future use).
func (c *Client) FetchWithHeaders(url string, headers map[string]string, skipCache bool) ([]byte, error) {
	return c.fetchWithHeaders(url, headers, skipCache)
}

// FetchJSON fetches and decodes JSON from the given URL.
// Returns ParseResult with successful events and individual parse errors.
func (c *Client) FetchJSON(url string, loc *time.Location) event.ParseResult {
	var result event.ParseResult

	// Fetch data (supports both HTTP and file:// URLs)
	body, err := c.fetch(url)
	if err != nil {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "JSON",
			Error:       err,
			RecoverType: "skipped",
		})
		return result
	}

	// Preprocess JSON to escape literal newlines in string values
	// Madrid's JSON sometimes contains unescaped newlines which are invalid JSON
	body = fixJSONNewlines(body)

	var response JSONResponse
	if err := json.Unmarshal(body, &response); err != nil {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "JSON",
			Error:       fmt.Errorf("decoding JSON: %w", err),
			RecoverType: "skipped",
		})
		return result
	}

	// Parse each event individually with error recovery
	for i, jsonEvent := range response.Graph {
		canonical, err := jsonEvent.ToCanonical(loc)
		if err != nil {
			// Log parse error but continue processing other events
			result.Errors = append(result.Errors, event.ParseError{
				Source:      "JSON",
				Index:       i,
				RawData:     fmt.Sprintf("ID=%s", jsonEvent.ID),
				Error:       err,
				RecoverType: "skipped",
			})
			continue
		}

		result.Events = append(result.Events, event.SourcedEvent{
			Event:  canonical,
			Source: "JSON",
		})
	}

	return result
}

// FetchXML fetches and decodes XML from the given URL.
// Returns ParseResult with successful events and individual parse errors.
func (c *Client) FetchXML(url string, loc *time.Location) event.ParseResult {
	var result event.ParseResult

	// Fetch data (supports both HTTP and file:// URLs)
	body, err := c.fetch(url)
	if err != nil {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "XML",
			Error:       err,
			RecoverType: "skipped",
		})
		return result
	}

	var response XMLResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "XML",
			Error:       fmt.Errorf("decoding XML: %w", err),
			RecoverType: "skipped",
		})
		return result
	}

	// Parse each event individually with error recovery
	for i, xmlEvent := range response.Events {
		canonical, err := xmlEvent.ToCanonical(loc)
		if err != nil {
			// Log parse error but continue processing other events
			result.Errors = append(result.Errors, event.ParseError{
				Source:      "XML",
				Index:       i,
				RawData:     fmt.Sprintf("ID=%s", xmlEvent.IDEvento),
				Error:       err,
				RecoverType: "skipped",
			})
			continue
		}

		result.Events = append(result.Events, event.SourcedEvent{
			Event:  canonical,
			Source: "XML",
		})
	}

	return result
}

// FetchCSV fetches and parses CSV from the given URL.
// Handles both semicolon and comma delimiters.
// Returns ParseResult with successful events and individual parse errors.
func (c *Client) FetchCSV(url string, loc *time.Location) event.ParseResult {
	var result event.ParseResult

	// Fetch data (supports both HTTP and file:// URLs)
	body, err := c.fetch(url)
	if err != nil {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "CSV",
			Error:       err,
			RecoverType: "skipped",
		})
		return result
	}

	// Try semicolon first (Madrid's preferred format)
	result = parseCSV(body, ';', loc)
	if len(result.Events) == 0 && len(result.Errors) > 0 {
		// Fall back to comma
		result = parseCSV(body, ',', loc)
	}

	return result
}

func parseCSV(data []byte, delimiter rune, loc *time.Location) event.ParseResult {
	var result event.ParseResult

	// Convert from ISO-8859-1/Windows-1252 to UTF-8
	// Madrid's CSV files use Windows-1252 encoding
	decoder := charmap.Windows1252.NewDecoder()
	utf8Data, err := io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
	if err != nil {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "CSV",
			Error:       fmt.Errorf("converting encoding: %w", err),
			RecoverType: "skipped",
		})
		return result
	}

	r := csv.NewReader(bytes.NewReader(utf8Data))
	r.Comma = delimiter
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "CSV",
			Error:       fmt.Errorf("parsing CSV: %w", err),
			RecoverType: "skipped",
		})
		return result
	}

	if len(records) < 2 {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "CSV",
			Error:       fmt.Errorf("CSV has no data rows"),
			RecoverType: "skipped",
		})
		return result
	}

	// Build header map
	headerMap := make(map[string]int)
	for i, col := range records[0] {
		headerMap[col] = i
	}

	// Validate that we have the expected ID-EVENTO column
	// (this helps detect wrong delimiter usage)
	if _, hasIDEvento := headerMap["ID-EVENTO"]; !hasIDEvento {
		result.Errors = append(result.Errors, event.ParseError{
			Source:      "CSV",
			Error:       fmt.Errorf("missing ID-EVENTO column (wrong delimiter?)"),
			RecoverType: "skipped",
		})
		return result
	}

	// Parse each row individually with error recovery
	for i, row := range records[1:] {
		csvEvent := parseCSVRow(row, headerMap)
		canonical, err := csvEvent.ToCanonical(loc)
		if err != nil {
			result.Errors = append(result.Errors, event.ParseError{
				Source:      "CSV",
				Index:       i,
				RawData:     fmt.Sprintf("ID=%s", csvEvent.IDEvento),
				Error:       err,
				RecoverType: "skipped",
			})
			continue
		}
		result.Events = append(result.Events, event.SourcedEvent{
			Event:  canonical,
			Source: "CSV",
		})
	}

	return result
}

// parseCSVRow extracts fields from a CSV row into a CSVEvent struct.
func parseCSVRow(row []string, headerMap map[string]int) CSVEvent {
	event := CSVEvent{
		IDEvento:          getField(row, headerMap, "ID-EVENTO"),
		Titulo:            getField(row, headerMap, "TITULO"),
		Fecha:             getField(row, headerMap, "FECHA"),
		FechaFin:          getField(row, headerMap, "FECHA-FIN"),
		Hora:              getField(row, headerMap, "HORA"),
		NombreInstalacion: getField(row, headerMap, "NOMBRE-INSTALACION"),
		Direccion:         getField(row, headerMap, "DIRECCION"),
		Distrito:          getField(row, headerMap, "DISTRITO-INSTALACION"),
		ContentURL:        getField(row, headerMap, "CONTENT-URL"),
		Descripcion:       getField(row, headerMap, "DESCRIPCION"),
	}

	// Parse coordinates
	if latStr := getField(row, headerMap, "LATITUD"); latStr != "" {
		fmt.Sscanf(latStr, "%f", &event.Latitud)
	}
	if lonStr := getField(row, headerMap, "LONGITUD"); lonStr != "" {
		fmt.Sscanf(lonStr, "%f", &event.Longitud)
	}

	return event
}

func getField(row []string, headerMap map[string]int, fieldName string) string {
	idx, ok := headerMap[fieldName]
	if !ok || idx >= len(row) {
		return ""
	}
	return row[idx]
}

// fetch retrieves data from a URL or local file.
// Supports both HTTP(S) URLs and file:// URLs.
// Uses HTTP caching with If-Modified-Since and throttling for respectful fetching.
func (c *Client) fetch(url string) ([]byte, error) {
	return c.fetchWithHeaders(url, nil, false)
}

// fetchWithHeaders retrieves data from a URL with custom HTTP headers.
// Supports both HTTP(S) URLs, file:// URLs, and synthetic URLs (cache-only).
// Uses HTTP caching with If-Modified-Since and throttling for respectful fetching.
// If skipCache is true, bypasses reading from cache (but still writes to cache for future use).
func (c *Client) fetchWithHeaders(url string, headers map[string]string, skipCache bool) ([]byte, error) {
	// Handle file:// URLs (no caching for local files)
	if strings.HasPrefix(url, "file://") {
		path := strings.TrimPrefix(url, "file://")
		return os.ReadFile(path)
	}

	// Handle synthetic URLs (cache-only, no network fetch)
	// Used for caching data under predictable keys (e.g., weather forecasts)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// Only check cache, never make network request
		cached, err := c.cache.Get(url)
		if err != nil || cached == nil {
			return nil, fmt.Errorf("cache miss for synthetic URL: %s", url)
		}
		c.auditor.Record(RequestRecord{
			URL:       url,
			Timestamp: time.Now(),
			CacheHit:  true,
		})
		return cached.Body, nil
	}

	// Strict test mode: block all external HTTP requests if PLAZAESPANA_NO_API is set
	// Allow localhost/127.0.0.1 for test servers (httptest)
	if os.Getenv("PLAZAESPANA_NO_API") != "" {
		if !strings.Contains(url, "://localhost") && !strings.Contains(url, "://127.0.0.1") {
			return nil, fmt.Errorf("BLOCKED: external API request to %s (PLAZAESPANA_NO_API is set - use mock servers in tests)", url)
		}
	}

	// Check cache first (unless skipCache is true)
	var cached *CacheEntry
	if !skipCache {
		var err error
		cached, err = c.cache.Get(url)
		if err != nil {
			// Cache read error - log but continue to fetch
			fmt.Fprintf(os.Stderr, "Warning: cache read error: %v\n", err)
		}

		if cached != nil {
			// Cache hit! Use cached data
			c.auditor.Record(RequestRecord{
				URL:       url,
				Timestamp: time.Now(),
				CacheHit:  true,
			})
			return cached.Body, nil
		}
	}

	// Cache miss - need to make HTTP request
	// Wait for throttle to allow request
	delay, err := c.throttle.Wait(url)
	if err != nil {
		return nil, fmt.Errorf("throttle error: %w", err)
	}

	if delay > 0 {
		// Log the delay so user knows why build is slow
		fmt.Fprintf(os.Stderr, "[%s] Waiting %v before fetching %s\n",
			c.config.Mode, delay.Round(time.Millisecond), url)
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	// Add custom headers (e.g., API keys)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add If-Modified-Since header if we have cached data (even if expired)
	if cached != nil && cached.LastModified != "" {
		req.Header.Set("If-Modified-Since", cached.LastModified)
	}

	// Make HTTP request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.auditor.Record(RequestRecord{
			URL:       url,
			Timestamp: time.Now(),
			CacheHit:  false,
			DelayMs:   delay.Milliseconds(),
			Error:     err.Error(),
		})
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified - use cached data
	if resp.StatusCode == http.StatusNotModified && cached != nil {
		c.auditor.Record(RequestRecord{
			URL:        url,
			Timestamp:  time.Now(),
			CacheHit:   true,
			StatusCode: 304,
			DelayMs:    delay.Milliseconds(),
		})
		return cached.Body, nil
	}

	// Check for rate limiting or errors
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusServiceUnavailable {
		c.auditor.Record(RequestRecord{
			URL:         url,
			Timestamp:   time.Now(),
			CacheHit:    false,
			StatusCode:  resp.StatusCode,
			DelayMs:     delay.Milliseconds(),
			RateLimited: true,
		})
		return nil, fmt.Errorf("HTTP %d (rate limited): %s", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		c.auditor.Record(RequestRecord{
			URL:        url,
			Timestamp:  time.Now(),
			CacheHit:   false,
			StatusCode: resp.StatusCode,
			DelayMs:    delay.Milliseconds(),
		})
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.auditor.Record(RequestRecord{
			URL:        url,
			Timestamp:  time.Now(),
			CacheHit:   false,
			StatusCode: resp.StatusCode,
			DelayMs:    delay.Milliseconds(),
			Error:      err.Error(),
		})
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Store in cache
	entry := CacheEntry{
		URL:          url,
		Body:         body,
		LastModified: resp.Header.Get("Last-Modified"),
		ETag:         resp.Header.Get("ETag"),
		StatusCode:   resp.StatusCode,
	}
	if err := c.cache.Set(entry); err != nil {
		// Log cache write error but don't fail the request
		fmt.Fprintf(os.Stderr, "Warning: cache write error: %v\n", err)
	}

	// Record successful fetch
	c.auditor.Record(RequestRecord{
		URL:        url,
		Timestamp:  time.Now(),
		CacheHit:   false,
		StatusCode: resp.StatusCode,
		DelayMs:    delay.Milliseconds(),
	})

	return body, nil
}

// fixJSONNewlines preprocesses JSON to escape literal newlines in string values.
// This handles Madrid's JSON which sometimes contains unescaped newlines.
func fixJSONNewlines(data []byte) []byte {
	var result bytes.Buffer
	inString := false
	escaped := false

	for i := 0; i < len(data); i++ {
		c := data[i]

		// Track if we're inside a string
		if c == '"' && !escaped {
			inString = !inString
			result.WriteByte(c)
			continue
		}

		// Track escape sequences
		if c == '\\' && !escaped {
			escaped = true
			result.WriteByte(c)
			continue
		}

		// If we're in a string and hit a literal newline, escape it
		if inString && !escaped && (c == '\n' || c == '\r') {
			if c == '\n' {
				result.WriteString("\\n")
			} else {
				result.WriteString("\\r")
			}
		} else {
			result.WriteByte(c)
		}

		escaped = false
	}

	return result.Bytes()
}
