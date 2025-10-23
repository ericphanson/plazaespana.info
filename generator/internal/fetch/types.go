package fetch

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
	"github.com/ericphanson/madrid-events/internal/validate"
)

// RawEvent represents a single event from Madrid's open data API.
// Field names match the upstream JSON/XML structure exactly.
type RawEvent struct {
	IDEvento          string  `json:"ID-EVENTO" xml:"ID-EVENTO"`
	Titulo            string  `json:"TITULO" xml:"TITULO"`
	Fecha             string  `json:"FECHA" xml:"FECHA"`
	FechaFin          string  `json:"FECHA-FIN" xml:"FECHA-FIN"`
	Hora              string  `json:"HORA" xml:"HORA"`
	NombreInstalacion string  `json:"NOMBRE-INSTALACION" xml:"NOMBRE-INSTALACION"`
	Direccion         string  `json:"DIRECCION" xml:"DIRECCION"`
	Lat               float64 `json:"COORDENADA-LATITUD" xml:"COORDENADA-LATITUD"`
	Lon               float64 `json:"COORDENADA-LONGITUD" xml:"COORDENADA-LONGITUD"`
	ContentURL        string  `json:"CONTENT-URL" xml:"CONTENT-URL"`
	Descripcion       string  `json:"DESCRIPCION" xml:"DESCRIPCION"`
}

// JSONEvent represents Madrid's JSON-LD event structure.
// Uses JSON-LD field names as mapped in @context.
type JSONEvent struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartTime   string  `json:"dtstart"`
	EndTime     string  `json:"dtend"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Location    string  `json:"event-location"`
	Link        string  `json:"link"`
}

// JSONResponse wraps the Madrid API JSON-LD structure.
type JSONResponse struct {
	Context interface{} `json:"@context"`
	Graph   []JSONEvent `json:"@graph"`
}

// parseJSONTime parses Madrid's JSON-LD datetime format.
// Format: "2025-10-25 19:00:00.0"
func parseJSONTime(dateStr string, loc *time.Location) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try formats in order of likelihood
	formats := []string{
		"2006-01-02 15:04:05.0", // With fractional seconds
		"2006-01-02 15:04:05",   // Without fractional seconds
		"2006-01-02",            // Date only
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.ParseInLocation(format, dateStr, loc)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("parsing date %q: %w", dateStr, lastErr)
}

// ToCanonical converts JSONEvent to CulturalEvent.
// Returns error if validation fails or required fields are missing.
func (e JSONEvent) ToCanonical(loc *time.Location) (event.CulturalEvent, error) {
	// Parse start time (format: "2025-10-25 19:00:00.0")
	startTime, err := parseJSONTime(e.StartTime, loc)
	if err != nil {
		return event.CulturalEvent{}, fmt.Errorf("parsing start time: %w", err)
	}

	// Parse end time (format: "2025-10-25 23:59:00.0")
	endTime, err := parseJSONTime(e.EndTime, loc)
	if err != nil {
		// End time is optional, set to zero time
		endTime = time.Time{}
	}

	canonical := event.CulturalEvent{
		ID:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		Latitude:    e.Latitude,
		Longitude:   e.Longitude,
		VenueName:   e.Location,
		DetailsURL:  e.Link,
		Sources:     []string{"JSON"},
	}

	// Sanitize and validate
	validate.SanitizeEvent(&canonical)
	if err := validate.ValidateEvent(canonical); err != nil {
		return event.CulturalEvent{}, err
	}

	return canonical, nil
}

// XMLEvent represents Madrid's XML event structure.
// XML uses different field names than CSV (e.g., FECHA-EVENTO vs FECHA).
type XMLEvent struct {
	IDEvento    string
	Titulo      string
	Descripcion string
	Fecha       string
	FechaFin    string
	Hora        string
	Latitud     float64
	Longitud    float64
	Instalacion string
	Direccion   string
	Distrito    string
	ContentURL  string
}

// xmlAtributo represents a single attribute in Madrid's XML structure.
type xmlAtributo struct {
	Nombre    string        `xml:"nombre,attr"`
	Value     string        `xml:",chardata"`
	Atributos []xmlAtributo `xml:"atributo"` // For nested LOCALIZACION
}

// UnmarshalXML implements custom XML unmarshaling for XMLEvent.
func (e *XMLEvent) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Parse the raw structure
	var raw struct {
		Atributos struct {
			Atributos []xmlAtributo `xml:"atributo"`
		} `xml:"atributos"`
	}

	if err := d.DecodeElement(&raw, &start); err != nil {
		return err
	}

	// Extract all attributes recursively
	attrs := make(map[string]string)
	for _, attr := range raw.Atributos.Atributos {
		extractAttributes(attr, attrs)
	}

	// Map to XMLEvent fields
	e.IDEvento = attrs["ID-EVENTO"]
	e.Titulo = attrs["TITULO"]
	e.Descripcion = attrs["DESCRIPCION"]
	e.Fecha = attrs["FECHA-EVENTO"]
	e.FechaFin = attrs["FECHA-FIN-EVENTO"]
	e.Hora = attrs["HORA-EVENTO"]
	e.Instalacion = attrs["NOMBRE-INSTALACION"]
	e.Direccion = attrs["DIRECCION-INSTALACION"]
	e.Distrito = attrs["DISTRITO"]
	e.ContentURL = attrs["CONTENT-URL"]

	// Parse coordinates
	if latStr := attrs["LATITUD"]; latStr != "" {
		fmt.Sscanf(latStr, "%f", &e.Latitud)
	}
	if lonStr := attrs["LONGITUD"]; lonStr != "" {
		fmt.Sscanf(lonStr, "%f", &e.Longitud)
	}

	return nil
}

// extractAttributes recursively extracts all nombre/value pairs from nested atributos.
func extractAttributes(attr xmlAtributo, result map[string]string) {
	// Store attribute value (skip container nodes like LOCALIZACION that have children)
	if attr.Nombre != "" && attr.Value != "" {
		result[attr.Nombre] = attr.Value
	}

	// Recursively extract nested attributes (including those inside LOCALIZACION)
	for _, nested := range attr.Atributos {
		extractAttributes(nested, result)
	}
}

// XMLResponse wraps the Madrid API XML structure.
type XMLResponse struct {
	XMLName xml.Name   `xml:"Contenidos"`
	Events  []XMLEvent `xml:"contenido"`
}

// ToCanonical converts XMLEvent to CulturalEvent.
// Returns error if validation fails or required fields are missing.
func (e XMLEvent) ToCanonical(loc *time.Location) (event.CulturalEvent, error) {
	// Import filter package for ParseEventDateTime
	// Parse times using filter.ParseEventDateTime (handles CSV format)
	startTime, err := parseXMLTime(e.Fecha, e.Hora, loc)
	if err != nil {
		return event.CulturalEvent{}, fmt.Errorf("parsing start time: %w", err)
	}

	// Parse end time (no HORA for end time)
	endTime, err := parseXMLTime(e.FechaFin, "", loc)
	if err != nil {
		// End time is optional, set to zero time
		endTime = time.Time{}
	}

	canonical := event.CulturalEvent{
		ID:          e.IDEvento,
		Title:       e.Titulo,
		Description: e.Descripcion,
		StartTime:   startTime,
		EndTime:     endTime,
		Latitude:    e.Latitud,
		Longitude:   e.Longitud,
		VenueName:   e.Instalacion,
		Address:     e.Direccion,
		Distrito:    e.Distrito,
		DetailsURL:  e.ContentURL,
		Sources:     []string{"XML"},
	}

	// Sanitize and validate
	validate.SanitizeEvent(&canonical)
	if err := validate.ValidateEvent(canonical); err != nil {
		return event.CulturalEvent{}, err
	}

	return canonical, nil
}

// parseXMLTime parses Madrid's XML datetime format.
// Format: "2025-10-25 00:00:00.0" for dates, "19:00" for time
func parseXMLTime(dateStr, timeStr string, loc *time.Location) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try formats in order of likelihood
	formats := []string{
		"2006-01-02 15:04:05.0", // With fractional seconds
		"2006-01-02 15:04:05",   // Without fractional seconds
		"2006-01-02",            // Date only
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.ParseInLocation(format, dateStr, loc)
		if err == nil {
			// If we have a separate time string, check if we should override the time portion
			// This handles XML where FECHA has "00:00:00" but HORA has the actual time
			if timeStr != "" {
				// Check if the parsed time is midnight (likely placeholder)
				if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
					// Parse time in HH:MM format
					timeOnlyFormat := "15:04"
					timeVal, timeErr := time.Parse(timeOnlyFormat, timeStr)
					if timeErr == nil {
						// Combine date with actual time from HORA field
						t = time.Date(t.Year(), t.Month(), t.Day(),
							timeVal.Hour(), timeVal.Minute(), 0, 0, loc)
					}
				}
			}
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("parsing date %q: %w", dateStr, lastErr)
}

// CSVEvent represents Madrid's CSV event structure.
// CSV uses same field names as XML but with different encoding.
type CSVEvent struct {
	IDEvento          string
	Titulo            string
	Descripcion       string
	Fecha             string
	FechaFin          string
	Hora              string
	Latitud           float64
	Longitud          float64
	NombreInstalacion string
	Direccion         string
	Distrito          string
	ContentURL        string
}

// ToCanonical converts CSVEvent to CulturalEvent.
// Returns error if validation fails or required fields are missing.
func (e CSVEvent) ToCanonical(loc *time.Location) (event.CulturalEvent, error) {
	// Parse times using parseXMLTime (CSV uses same format as XML)
	startTime, err := parseXMLTime(e.Fecha, e.Hora, loc)
	if err != nil {
		return event.CulturalEvent{}, fmt.Errorf("parsing start time: %w", err)
	}

	// Parse end time (no separate HORA for end time in CSV)
	endTime, err := parseXMLTime(e.FechaFin, "", loc)
	if err != nil {
		// End time is optional, set to zero time
		endTime = time.Time{}
	}

	canonical := event.CulturalEvent{
		ID:          e.IDEvento,
		Title:       e.Titulo,
		Description: e.Descripcion,
		StartTime:   startTime,
		EndTime:     endTime,
		Latitude:    e.Latitud,
		Longitude:   e.Longitud,
		VenueName:   e.NombreInstalacion,
		Address:     e.Direccion,
		Distrito:    e.Distrito,
		DetailsURL:  e.ContentURL,
		Sources:     []string{"CSV"},
	}

	// Sanitize and validate
	validate.SanitizeEvent(&canonical)
	if err := validate.ValidateEvent(canonical); err != nil {
		return event.CulturalEvent{}, err
	}

	return canonical, nil
}
