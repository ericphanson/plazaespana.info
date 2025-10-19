package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yourusername/madrid-events/internal/fetch"
)

func TestManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	events := []fetch.RawEvent{
		{IDEvento: "SNAP-001", Titulo: "Snapshot Event"},
		{IDEvento: "SNAP-002", Titulo: "Another Event"},
	}

	// Save snapshot
	err := mgr.SaveSnapshot(events)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Verify file exists
	snapshotPath := filepath.Join(tmpDir, "last_success.json")
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		t.Fatal("Snapshot file was not created")
	}

	// Load snapshot
	loaded, err := mgr.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(loaded))
	}

	if loaded[0].IDEvento != "SNAP-001" {
		t.Errorf("Expected IDEvento 'SNAP-001', got '%s'", loaded[0].IDEvento)
	}
}

func TestManager_LoadSnapshot_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.LoadSnapshot()
	if err == nil {
		t.Error("Expected error when loading non-existent snapshot")
	}
}
