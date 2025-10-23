package fetch

import (
	"os"
	"path/filepath"
	"testing"
)

// getFixturePath returns the absolute file:// URL for a fixture file
func getFixturePath(t *testing.T, filename string) string {
	// Try multiple possible paths since go test working directory can vary
	possiblePaths := []string{
		filepath.Join("testdata", "fixtures", filename),             // From project root
		filepath.Join("..", "..", "testdata", "fixtures", filename), // From package dir
	}

	for _, relPath := range possiblePaths {
		absPath, err := filepath.Abs(relPath)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return "file://" + absPath
		}
	}

	// If we get here, no fixtures found - skip test
	t.Skipf("Fixture file %s not found", filename)
	return ""
}
