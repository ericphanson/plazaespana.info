package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
)

// AuditFile represents the complete audit trail for a single build.
// Contains all events from both pipelines with full filter details.
type AuditFile struct {
	BuildTime     time.Time `json:"build_time"`
	BuildDuration float64   `json:"build_duration_seconds"`
	TotalEvents   int       `json:"total_events"`

	CulturalEvents AuditPipeline    `json:"cultural_events"`
	CityEvents     AuditPipeline    `json:"city_events"`
	ParseErrors    ParseErrorsAudit `json:"parse_errors"` // NEW: Track parse failures
}

// AuditPipeline tracks all events and filtering decisions for one pipeline.
type AuditPipeline struct {
	Total           int               `json:"total"`
	Kept            int               `json:"kept"`
	Filtered        int               `json:"filtered"`
	FilterBreakdown map[string]int    `json:"filter_breakdown"`
	Events          []json.RawMessage `json:"events"` // Raw JSON to support different event types
}

// SaveAuditJSON exports complete audit trail to JSON file.
// Includes all events (kept + filtered) with filter decisions and parse errors.
func SaveAuditJSON(
	culturalEvents []event.CulturalEvent,
	cityEvents []event.CityEvent,
	culturalParseErrors []event.ParseError,
	cityParseErrors []event.ParseError,
	path string,
	buildTime time.Time,
	duration time.Duration,
) error {
	// Process cultural events
	culturalPipeline, err := processCulturalEvents(culturalEvents)
	if err != nil {
		return fmt.Errorf("processing cultural events: %w", err)
	}

	// Process city events
	cityPipeline, err := processCityEvents(cityEvents)
	if err != nil {
		return fmt.Errorf("processing city events: %w", err)
	}

	// Process parse errors
	parseErrorsAudit := processParseErrors(culturalParseErrors, cityParseErrors)

	// Build audit file
	audit := AuditFile{
		BuildTime:      buildTime,
		BuildDuration:  duration.Seconds(),
		TotalEvents:    len(culturalEvents) + len(cityEvents),
		CulturalEvents: culturalPipeline,
		CityEvents:     cityPipeline,
		ParseErrors:    parseErrorsAudit,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling audit JSON: %w", err)
	}

	// Write atomically (temp file + rename)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating audit directory: %w", err)
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp audit file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("renaming audit file: %w", err)
	}

	return nil
}

// processCulturalEvents analyzes cultural events and builds pipeline stats.
func processCulturalEvents(events []event.CulturalEvent) (AuditPipeline, error) {
	pipeline := AuditPipeline{
		Total:           len(events),
		FilterBreakdown: make(map[string]int),
		Events:          make([]json.RawMessage, len(events)),
	}

	for i, evt := range events {
		if evt.FilterResult.Kept {
			pipeline.Kept++
		} else {
			pipeline.Filtered++
		}

		// Count filter reasons
		if evt.FilterResult.FilterReason != "" {
			pipeline.FilterBreakdown[evt.FilterResult.FilterReason]++
		}

		// Marshal event to JSON
		data, err := json.Marshal(evt)
		if err != nil {
			return pipeline, fmt.Errorf("marshaling cultural event %s: %w", evt.ID, err)
		}
		pipeline.Events[i] = data
	}

	return pipeline, nil
}

// processCityEvents analyzes city events and builds pipeline stats.
func processCityEvents(events []event.CityEvent) (AuditPipeline, error) {
	pipeline := AuditPipeline{
		Total:           len(events),
		FilterBreakdown: make(map[string]int),
		Events:          make([]json.RawMessage, len(events)),
	}

	for i, evt := range events {
		if evt.FilterResult.Kept {
			pipeline.Kept++
		} else {
			pipeline.Filtered++
		}

		// Count filter reasons
		if evt.FilterResult.FilterReason != "" {
			pipeline.FilterBreakdown[evt.FilterResult.FilterReason]++
		}

		// Marshal event to JSON
		data, err := json.Marshal(evt)
		if err != nil {
			return pipeline, fmt.Errorf("marshaling city event %s: %w", evt.ID, err)
		}
		pipeline.Events[i] = data
	}

	return pipeline, nil
}

// processParseErrors converts parse errors from both pipelines into audit format.
func processParseErrors(culturalErrors []event.ParseError, cityErrors []event.ParseError) ParseErrorsAudit {
	// Convert cultural parse errors
	culturalAudit := make([]AuditParseError, 0, len(culturalErrors))
	for _, e := range culturalErrors {
		culturalAudit = append(culturalAudit, AuditParseError{
			Source:      e.Source,
			Index:       e.Index,
			RawData:     e.RawData,
			Error:       e.Error.Error(),
			RecoverType: e.RecoverType,
		})
	}

	// Convert city parse errors
	cityAudit := make([]AuditParseError, 0, len(cityErrors))
	for _, e := range cityErrors {
		cityAudit = append(cityAudit, AuditParseError{
			Source:      e.Source,
			Index:       e.Index,
			RawData:     e.RawData,
			Error:       e.Error.Error(),
			RecoverType: e.RecoverType,
		})
	}

	return ParseErrorsAudit{
		CulturalErrors: culturalAudit,
		CityErrors:     cityAudit,
		TotalErrors:    len(culturalErrors) + len(cityErrors),
	}
}
