package fetch

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
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
func (c *Client) FetchJSON(url string) (*JSONResponse, error) {
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

	var result JSONResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}

	return &result, nil
}

// XMLResponse wraps the Madrid API XML structure.
type XMLResponse struct {
	XMLName xml.Name   `xml:"response"`
	Events  []RawEvent `xml:"event"`
}

// FetchXML fetches and decodes XML from the given URL.
func (c *Client) FetchXML(url string) ([]RawEvent, error) {
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

	var result XMLResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding XML: %w", err)
	}

	return result.Events, nil
}

// FetchCSV fetches and parses CSV from the given URL.
// Handles both semicolon and comma delimiters.
func (c *Client) FetchCSV(url string) ([]RawEvent, error) {
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

	// Try semicolon first (Madrid's preferred format)
	events, err := parseCSV(body, ';')
	if err != nil || len(events) == 0 {
		// Fall back to comma
		events, err = parseCSV(body, ',')
	}

	return events, err
}

func parseCSV(data []byte, delimiter rune) ([]RawEvent, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = delimiter
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV has no data rows")
	}

	// Build header map
	headerMap := make(map[string]int)
	for i, col := range records[0] {
		headerMap[col] = i
	}

	// Validate that we have the expected ID-EVENTO column
	// (this helps detect wrong delimiter usage)
	if _, hasIDEvento := headerMap["ID-EVENTO"]; !hasIDEvento {
		return nil, fmt.Errorf("missing ID-EVENTO column (wrong delimiter?)")
	}

	var events []RawEvent
	for i := 1; i < len(records); i++ {
		row := records[i]
		event := RawEvent{
			IDEvento:          getField(row, headerMap, "ID-EVENTO"),
			Titulo:            getField(row, headerMap, "TITULO"),
			Fecha:             getField(row, headerMap, "FECHA"),
			FechaFin:          getField(row, headerMap, "FECHA-FIN"),
			Hora:              getField(row, headerMap, "HORA"),
			NombreInstalacion: getField(row, headerMap, "NOMBRE-INSTALACION"),
			Direccion:         getField(row, headerMap, "DIRECCION"),
			ContentURL:        getField(row, headerMap, "CONTENT-URL"),
			Descripcion:       getField(row, headerMap, "DESCRIPCION"),
		}

		// Parse coordinates
		if latStr := getField(row, headerMap, "COORDENADA-LATITUD"); latStr != "" {
			fmt.Sscanf(latStr, "%f", &event.Lat)
		}
		if lonStr := getField(row, headerMap, "COORDENADA-LONGITUD"); lonStr != "" {
			fmt.Sscanf(lonStr, "%f", &event.Lon)
		}

		events = append(events, event)
	}

	return events, nil
}

func getField(row []string, headerMap map[string]int, fieldName string) string {
	idx, ok := headerMap[fieldName]
	if !ok || idx >= len(row) {
		return ""
	}
	return row[idx]
}
