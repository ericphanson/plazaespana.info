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

	"github.com/ericphanson/madrid-events/internal/event"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Client wraps an HTTP client for fetching Madrid event data.
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a fetch client with the given timeout.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent: "madrid-events-site-generator/1.0 (https://github.com/ericphanson/madrid-events)",
	}
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
func (c *Client) fetch(url string) ([]byte, error) {
	// Handle file:// URLs
	if strings.HasPrefix(url, "file://") {
		path := strings.TrimPrefix(url, "file://")
		return os.ReadFile(path)
	}

	// Handle HTTP(S) URLs
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

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
