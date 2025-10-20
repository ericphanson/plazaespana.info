package event

import "time"

// CanonicalEvent represents an event in our internal format.
// All parsers convert to this structure.
type CanonicalEvent struct {
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

	// Metadata
	DetailsURL string

	// Source tracking
	Sources []string // ["JSON", "XML", "CSV"]
}

// SourcedEvent wraps an event with its source.
type SourcedEvent struct {
	Event  CanonicalEvent
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
