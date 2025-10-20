package fetch

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
)

// EsmadridServiceList represents the root element of the ESMadrid XML feed.
type EsmadridServiceList struct {
	XMLName  xml.Name          `xml:"serviceList"`
	Services []EsmadridService `xml:"service"`
}

// EsmadridService represents a single service/event from the ESMadrid XML feed.
// This matches the actual XML structure from esmadrid.com/agenda.
type EsmadridService struct {
	ID         string `xml:"id,attr"`
	UpdateDate string `xml:"fechaActualizacion,attr"`

	// Basic data fields
	Name      string `xml:"basicData>name"`
	Title     string `xml:"basicData>title"`
	Body      string `xml:"basicData>body"`
	Web       string `xml:"basicData>web"`
	VenueID   string `xml:"basicData>idrt"`
	VenueName string `xml:"basicData>nombrert"`

	// Geo data fields
	Address   string `xml:"geoData>address"`
	Latitude  string `xml:"geoData>latitude"`
	Longitude string `xml:"geoData>longitude"`

	// Multimedia
	ImageURL string `xml:"multimedia>media>url"`

	// Extra data (nested structure - requires custom parsing)
	Category    string // Extracted from extradata>categorias>categoria>item[@name="Categoria"]
	Subcategory string // Extracted from extradata>categorias>categoria>subcategorias>subcategoria>item[@name="SubCategoria"]
	Price       string // Extracted from extradata>item[@name="Servicios de pago"]
	StartDate   string // Extracted from extradata>fechas>rango>inicio
	EndDate     string // Extracted from extradata>fechas>rango>fin

	// Raw extradata for custom parsing
	RawExtradata esmadridExtradata `xml:"extradata"`
}

// esmadridExtradata holds the raw extradata structure for custom parsing.
type esmadridExtradata struct {
	Items      []esmadridItem     `xml:"item"`
	Categorias esmadridCategorias `xml:"categorias"`
	Fechas     esmadridFechas     `xml:"fechas"`
}

// esmadridItem represents a name-value pair in extradata.
type esmadridItem struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// esmadridCategorias represents the categorias structure.
type esmadridCategorias struct {
	Categoria esmadridCategoria `xml:"categoria"`
}

// esmadridCategoria represents a single categoria.
type esmadridCategoria struct {
	Items         []esmadridItem        `xml:"item"`
	Subcategorias esmadridSubcategorias `xml:"subcategorias"`
}

// esmadridSubcategorias represents the subcategorias structure.
type esmadridSubcategorias struct {
	Subcategoria esmadridSubcategoria `xml:"subcategoria"`
}

// esmadridSubcategoria represents a single subcategoria.
type esmadridSubcategoria struct {
	Items []esmadridItem `xml:"item"`
}

// esmadridFechas represents the fechas structure.
type esmadridFechas struct {
	Rango esmadridRango `xml:"rango"`
}

// esmadridRango represents a date range.
type esmadridRango struct {
	Inicio string `xml:"inicio"`
	Fin    string `xml:"fin"`
}

// UnmarshalXML implements custom XML unmarshaling for EsmadridService.
// This extracts nested extradata fields into top-level struct fields.
func (e *EsmadridService) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Create a temporary struct with the same structure
	type Alias EsmadridService
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	// Decode into the alias
	if err := d.DecodeElement(aux, &start); err != nil {
		return err
	}

	// Extract category and subcategory from nested structure
	for _, item := range e.RawExtradata.Categorias.Categoria.Items {
		if item.Name == "Categoria" {
			e.Category = unescapeHTML(item.Value)
		}
	}

	for _, item := range e.RawExtradata.Categorias.Categoria.Subcategorias.Subcategoria.Items {
		if item.Name == "SubCategoria" {
			e.Subcategory = unescapeHTML(item.Value)
		}
	}

	// Extract price from items
	for _, item := range e.RawExtradata.Items {
		if item.Name == "Servicios de pago" {
			e.Price = item.Value
		}
	}

	// Extract date range
	e.StartDate = e.RawExtradata.Fechas.Rango.Inicio
	e.EndDate = e.RawExtradata.Fechas.Rango.Fin

	// Unescape HTML entities in CDATA fields
	e.Name = unescapeHTML(e.Name)
	e.Title = unescapeHTML(e.Title)
	e.VenueName = unescapeHTML(e.VenueName)

	return nil
}

// unescapeHTML unescapes HTML entities in a string.
// ESMadrid uses HTML entities like &aacute; in CDATA sections.
func unescapeHTML(s string) string {
	return html.UnescapeString(s)
}

// ToCityEvent converts an EsmadridService to a CityEvent.
// Returns error if required fields are missing or invalid.
func (e EsmadridService) ToCityEvent() (*event.CityEvent, error) {
	// Parse coordinates
	lat, err := parseFloat(e.Latitude)
	if err != nil {
		// Coordinates are optional, default to 0
		lat = 0.0
	}

	lon, err := parseFloat(e.Longitude)
	if err != nil {
		// Coordinates are optional, default to 0
		lon = 0.0
	}

	// Parse dates (format: DD/MM/YYYY)
	startDate, err := parseEsmadridDate(e.StartDate)
	if err != nil {
		return nil, fmt.Errorf("parsing start date %q: %w", e.StartDate, err)
	}

	endDate, err := parseEsmadridDate(e.EndDate)
	if err != nil {
		// End date is optional, default to start date
		endDate = startDate
	}

	// Use Title if present, otherwise fall back to Name
	title := e.Title
	if title == "" {
		title = e.Name
	}

	cityEvent := &event.CityEvent{
		ID:          e.ID,
		Title:       title,
		Description: e.Body,
		StartDate:   startDate,
		EndDate:     endDate,
		Venue:       e.VenueName,
		Address:     e.Address,
		Latitude:    lat,
		Longitude:   lon,
		Category:    e.Category,
		Subcategory: e.Subcategory,
		WebURL:      e.Web,
		ImageURL:    e.ImageURL,
		Price:       e.Price,
	}

	return cityEvent, nil
}

// parseFloat parses a string to float64, handling empty strings.
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0.0, fmt.Errorf("empty string")
	}
	return strconv.ParseFloat(s, 64)
}

// parseEsmadridDate parses ESMadrid's date format (DD/MM/YYYY) to time.Time.
// Times are set to midnight in Europe/Madrid timezone.
func parseEsmadridDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Parse DD/MM/YYYY format
	parts := strings.Split(dateStr, "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format %q, expected DD/MM/YYYY", dateStr)
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day %q: %w", parts[0], err)
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month %q: %w", parts[1], err)
	}

	year, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year %q: %w", parts[2], err)
	}

	// Load Europe/Madrid timezone
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		// Fallback to UTC if timezone loading fails
		loc = time.UTC
	}

	// Create time at midnight in Madrid timezone
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc), nil
}

// FetchEsmadridEvents fetches and parses ESMadrid events from the given URL.
// Returns a slice of EsmadridService structs or an error if fetching/parsing fails.
func FetchEsmadridEvents(url string) ([]EsmadridService, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set User-Agent header (matching pattern from client.go)
	req.Header.Set("User-Agent", "madrid-events-site-generator/1.0 (https://github.com/ericphanson/madrid-events)")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Parse XML
	var serviceList EsmadridServiceList
	if err := xml.Unmarshal(body, &serviceList); err != nil {
		return nil, fmt.Errorf("parsing XML: %w", err)
	}

	return serviceList.Services, nil
}
