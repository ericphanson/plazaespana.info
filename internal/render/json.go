package render

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONRenderer renders events to JSON.
type JSONRenderer struct{}

// NewJSONRenderer creates a JSON renderer.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// Render generates JSON output and writes it atomically to outputPath.
// culturalEvents: events from datos.madrid.es
// cityEvents: events from esmadrid.com
// updateTime: timestamp when the data was generated
func (r *JSONRenderer) Render(culturalEvents, cityEvents []JSONEvent, updateTime time.Time, outputPath string) error {
	// Build the structured output
	output := JSONOutput{
		CulturalEvents: culturalEvents,
		CityEvents:     cityEvents,
		Meta: JSONMeta{
			UpdateTime:    updateTime.Format(time.RFC3339),
			TotalCultural: len(culturalEvents),
			TotalCity:     len(cityEvents),
		},
	}

	// Ensure empty arrays instead of null in JSON
	if output.CulturalEvents == nil {
		output.CulturalEvents = []JSONEvent{}
	}
	if output.CityEvents == nil {
		output.CityEvents = []JSONEvent{}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	// Atomic write: temp file + rename
	tmpPath := outputPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("renaming output: %w", err)
	}

	return nil
}
