package event

import "time"

// CulturalEvent represents an event in our internal format.
// All parsers convert to this structure.
type CulturalEvent struct {
	// Core fields
	ID          string
	Title       string
	Description string

	// Time
	StartTime time.Time
	EndTime   time.Time

	// Location
	Latitude  float64
	Longitude float64
	VenueName string
	Address   string
	Distrito  string // District where event takes place (e.g. "CENTRO", "MONCLOA-ARAVACA")

	// Metadata
	DetailsURL string

	// Source tracking
	Sources []string // ["JSON", "XML", "CSV"]
}

// EventType returns the type of this event.
func (e CulturalEvent) EventType() string {
	return "cultural"
}

// SourcedEvent wraps an event with its source.
type SourcedEvent struct {
	Event  CulturalEvent
	Source string // "JSON", "XML", or "CSV"
}

// ParseResult tracks both successful parses and failures.
type ParseResult struct {
	Events []SourcedEvent
	Errors []ParseError
}

// ParseError records a single event that failed to parse.
type ParseError struct {
	Source      string // "JSON", "XML", "CSV"
	Index       int    // Position in source data
	RawData     string // Snippet of problematic data
	Error       error  // What went wrong
	RecoverType string // "skipped", "partial", "defaulted"
}
