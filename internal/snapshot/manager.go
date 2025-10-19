package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourusername/madrid-events/internal/fetch"
)

// Manager handles saving and loading event snapshots for fallback resilience.
type Manager struct {
	dataDir string
}

// NewManager creates a snapshot manager for the given data directory.
func NewManager(dataDir string) *Manager {
	return &Manager{dataDir: dataDir}
}

// SaveSnapshot saves events to last_success.json.
func (m *Manager) SaveSnapshot(events []fetch.RawEvent) error {
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	snapshotPath := filepath.Join(m.dataDir, "last_success.json")
	tmpPath := snapshotPath + ".tmp"

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding snapshot: %w", err)
	}

	// Atomic write: write to temp file, then rename
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp snapshot: %w", err)
	}

	if err := os.Rename(tmpPath, snapshotPath); err != nil {
		return fmt.Errorf("renaming snapshot: %w", err)
	}

	return nil
}

// LoadSnapshot loads events from last_success.json.
func (m *Manager) LoadSnapshot() ([]fetch.RawEvent, error) {
	snapshotPath := filepath.Join(m.dataDir, "last_success.json")

	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}

	var events []fetch.RawEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("decoding snapshot: %w", err)
	}

	return events, nil
}
