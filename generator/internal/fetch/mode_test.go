package fetch

import (
	"testing"
	"time"
)

func TestDefaultProductionConfig(t *testing.T) {
	config := DefaultProductionConfig()

	if config.Mode != ProductionMode {
		t.Errorf("Mode = %v, want %v", config.Mode, ProductionMode)
	}
	if config.CacheTTL != 30*time.Minute {
		t.Errorf("CacheTTL = %v, want 30m", config.CacheTTL)
	}
	if config.MinDelay != 2*time.Second {
		t.Errorf("MinDelay = %v, want 2s", config.MinDelay)
	}
	if config.MaxRequestRate != 1 {
		t.Errorf("MaxRequestRate = %d, want 1", config.MaxRequestRate)
	}
	if config.TimeWindow != 1*time.Hour {
		t.Errorf("TimeWindow = %v, want 1h", config.TimeWindow)
	}
}

func TestDefaultDevelopmentConfig(t *testing.T) {
	config := DefaultDevelopmentConfig()

	if config.Mode != DevelopmentMode {
		t.Errorf("Mode = %v, want %v", config.Mode, DevelopmentMode)
	}
	if config.CacheTTL != 1*time.Hour {
		t.Errorf("CacheTTL = %v, want 1h", config.CacheTTL)
	}
	if config.MinDelay != 5*time.Second {
		t.Errorf("MinDelay = %v, want 5s", config.MinDelay)
	}
	if config.MaxRequestRate != 1 {
		t.Errorf("MaxRequestRate = %d, want 1", config.MaxRequestRate)
	}
	if config.TimeWindow != 5*time.Minute {
		t.Errorf("TimeWindow = %v, want 5m", config.TimeWindow)
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input string
		want  ClientMode
	}{
		{"production", ProductionMode},
		{"prod", ProductionMode},
		{"development", DevelopmentMode},
		{"dev", DevelopmentMode},
		{"invalid", ProductionMode}, // Safe default
		{"", ProductionMode},        // Safe default
	}

	for _, tt := range tests {
		got := ParseMode(tt.input)
		if got != tt.want {
			t.Errorf("ParseMode(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
