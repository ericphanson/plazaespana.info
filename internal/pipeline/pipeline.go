package pipeline

import (
	"fmt"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
	"github.com/ericphanson/madrid-events/internal/fetch"
)

// Pipeline coordinates parallel data source fetching.
type Pipeline struct {
	jsonURL string
	xmlURL  string
	csvURL  string
	client  *fetch.Client
	loc     *time.Location
}

// NewPipeline creates a new pipeline with the given URLs and client.
func NewPipeline(jsonURL, xmlURL, csvURL string, client *fetch.Client, loc *time.Location) *Pipeline {
	return &Pipeline{
		jsonURL: jsonURL,
		xmlURL:  xmlURL,
		csvURL:  csvURL,
		client:  client,
		loc:     loc,
	}
}

// PipelineResult tracks events from all sources.
type PipelineResult struct {
	JSONEvents []event.SourcedEvent
	XMLEvents  []event.SourcedEvent
	CSVEvents  []event.SourcedEvent

	JSONErrors []event.ParseError
	XMLErrors  []event.ParseError
	CSVErrors  []event.ParseError
}

// FetchAll fetches from all three sources sequentially.
// Each source is isolated - errors in one don't affect others.
func (p *Pipeline) FetchAll() PipelineResult {
	var result PipelineResult

	// Fetch JSON (isolated - errors captured, don't crash)
	result.JSONEvents, result.JSONErrors = p.fetchJSONIsolated()

	// Fetch XML (isolated - JSON failure doesn't prevent this)
	result.XMLEvents, result.XMLErrors = p.fetchXMLIsolated()

	// Fetch CSV (isolated - previous failures don't prevent this)
	result.CSVEvents, result.CSVErrors = p.fetchCSVIsolated()

	return result
}

// fetchJSONIsolated fetches JSON with panic recovery.
func (p *Pipeline) fetchJSONIsolated() (events []event.SourcedEvent, errors []event.ParseError) {
	defer func() {
		if r := recover(); r != nil {
			errors = append(errors, event.ParseError{
				Source:      "JSON",
				Error:       fmt.Errorf("JSON fetch panic: %v", r),
				RecoverType: "skipped",
			})
		}
	}()

	result := p.client.FetchJSON(p.jsonURL, p.loc)
	return result.Events, result.Errors
}

// fetchXMLIsolated fetches XML with panic recovery.
func (p *Pipeline) fetchXMLIsolated() (events []event.SourcedEvent, errors []event.ParseError) {
	defer func() {
		if r := recover(); r != nil {
			errors = append(errors, event.ParseError{
				Source:      "XML",
				Error:       fmt.Errorf("XML fetch panic: %v", r),
				RecoverType: "skipped",
			})
		}
	}()

	result := p.client.FetchXML(p.xmlURL, p.loc)
	return result.Events, result.Errors
}

// fetchCSVIsolated fetches CSV with panic recovery.
func (p *Pipeline) fetchCSVIsolated() (events []event.SourcedEvent, errors []event.ParseError) {
	defer func() {
		if r := recover(); r != nil {
			errors = append(errors, event.ParseError{
				Source:      "CSV",
				Error:       fmt.Errorf("CSV fetch panic: %v", r),
				RecoverType: "skipped",
			})
		}
	}()

	result := p.client.FetchCSV(p.csvURL, p.loc)
	return result.Events, result.Errors
}

// Merge combines events from all sources and deduplicates.
// Events found in multiple sources will have all sources tracked.
func (p *Pipeline) Merge(result PipelineResult) []event.CanonicalEvent {
	// Combine all events
	var all []event.SourcedEvent
	all = append(all, result.JSONEvents...)
	all = append(all, result.XMLEvents...)
	all = append(all, result.CSVEvents...)

	// Deduplicate by ID, tracking sources
	seen := make(map[string]*event.CanonicalEvent)

	for _, sourced := range all {
		if existing, found := seen[sourced.Event.ID]; found {
			// Event already exists, add this source
			existing.Sources = append(existing.Sources, sourced.Source)

			// Merge distrito if the new source has it but existing doesn't
			if existing.Distrito == "" && sourced.Event.Distrito != "" {
				existing.Distrito = sourced.Event.Distrito
			}

			// Merge other missing fields as needed
			if existing.VenueName == "" && sourced.Event.VenueName != "" {
				existing.VenueName = sourced.Event.VenueName
			}
			if existing.Address == "" && sourced.Event.Address != "" {
				existing.Address = sourced.Event.Address
			}
			if existing.Description == "" && sourced.Event.Description != "" {
				existing.Description = sourced.Event.Description
			}
			if existing.Latitude == 0 && sourced.Event.Latitude != 0 {
				existing.Latitude = sourced.Event.Latitude
			}
			if existing.Longitude == 0 && sourced.Event.Longitude != 0 {
				existing.Longitude = sourced.Event.Longitude
			}
		} else {
			// New event
			evt := sourced.Event
			seen[evt.ID] = &evt
		}
	}

	// Convert map to slice
	merged := make([]event.CanonicalEvent, 0, len(seen))
	for _, evt := range seen {
		merged = append(merged, *evt)
	}

	return merged
}
