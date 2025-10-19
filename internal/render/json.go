package render

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONRenderer renders events to JSON.
type JSONRenderer struct{}

// NewJSONRenderer creates a JSON renderer.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// Render generates JSON output and writes it atomically to outputPath.
func (r *JSONRenderer) Render(events []JSONEvent, outputPath string) error {
	data, err := json.MarshalIndent(events, "", "  ")
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
