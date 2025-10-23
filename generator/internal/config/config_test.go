package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	validTOML := `
[cultural_events]
json_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json"
xml_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml"
csv_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv"

[city_events]
xml_url = "https://www.esmadrid.com/opendata/agenda_v1_es.xml"

[filter]
latitude = 40.42338
longitude = -3.71217
radius_km = 0.35
distritos = ["CENTRO", "MONCLOA-ARAVACA"]
past_events_weeks = 2

[output]
html_path = "public/index.html"
json_path = "public/events.json"

[snapshot]
data_dir = "data"

[server]
port = 8080

[weather]
api_key_env = "AEMET_API_KEY"
municipality_code = "28079"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(validTOML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify CulturalEvents
	if cfg.CulturalEvents.JSONURL != "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json" {
		t.Errorf("CulturalEvents.JSONURL = %q, want %q", cfg.CulturalEvents.JSONURL, "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json")
	}
	if cfg.CulturalEvents.XMLURL != "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml" {
		t.Errorf("CulturalEvents.XMLURL = %q, want %q", cfg.CulturalEvents.XMLURL, "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml")
	}
	if cfg.CulturalEvents.CSVURL != "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv" {
		t.Errorf("CulturalEvents.CSVURL = %q, want %q", cfg.CulturalEvents.CSVURL, "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv")
	}

	// Verify CityEvents
	if cfg.CityEvents.XMLURL != "https://www.esmadrid.com/opendata/agenda_v1_es.xml" {
		t.Errorf("CityEvents.XMLURL = %q, want %q", cfg.CityEvents.XMLURL, "https://www.esmadrid.com/opendata/agenda_v1_es.xml")
	}

	// Verify Filter
	if cfg.Filter.Latitude != 40.42338 {
		t.Errorf("Filter.Latitude = %f, want %f", cfg.Filter.Latitude, 40.42338)
	}
	if cfg.Filter.Longitude != -3.71217 {
		t.Errorf("Filter.Longitude = %f, want %f", cfg.Filter.Longitude, -3.71217)
	}
	if cfg.Filter.RadiusKm != 0.35 {
		t.Errorf("Filter.RadiusKm = %f, want %f", cfg.Filter.RadiusKm, 0.35)
	}
	if len(cfg.Filter.Distritos) != 2 {
		t.Errorf("Filter.Distritos length = %d, want 2", len(cfg.Filter.Distritos))
	}
	if cfg.Filter.Distritos[0] != "CENTRO" || cfg.Filter.Distritos[1] != "MONCLOA-ARAVACA" {
		t.Errorf("Filter.Distritos = %v, want [CENTRO MONCLOA-ARAVACA]", cfg.Filter.Distritos)
	}
	if cfg.Filter.PastEventsWeeks != 2 {
		t.Errorf("Filter.PastEventsWeeks = %d, want 2", cfg.Filter.PastEventsWeeks)
	}

	// Verify Output
	if cfg.Output.HTMLPath != "public/index.html" {
		t.Errorf("Output.HTMLPath = %q, want %q", cfg.Output.HTMLPath, "public/index.html")
	}
	if cfg.Output.JSONPath != "public/events.json" {
		t.Errorf("Output.JSONPath = %q, want %q", cfg.Output.JSONPath, "public/events.json")
	}

	// Verify Snapshot
	if cfg.Snapshot.DataDir != "data" {
		t.Errorf("Snapshot.DataDir = %q, want %q", cfg.Snapshot.DataDir, "data")
	}

	// Verify Server
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.toml")
	if err == nil {
		t.Fatal("Load() succeeded, want error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Load() error = %v, want error containing 'failed to read config file'", err)
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	invalidTOML := `
[cultural_events
json_url = "invalid syntax
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")
	if err := os.WriteFile(configPath, []byte(invalidTOML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() succeeded, want error for invalid TOML")
	}
	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("Load() error = %v, want error containing 'failed to parse config file'", err)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		CulturalEvents: CulturalEventsConfig{
			JSONURL: "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
			XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
			CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
		},
		CityEvents: CityEventsConfig{
			XMLURL: "https://www.esmadrid.com/opendata/agenda_v1_es.xml",
		},
		Filter: FilterConfig{
			Latitude:        40.42338,
			Longitude:       -3.71217,
			RadiusKm:        0.35,
			Distritos:       []string{"CENTRO"},
			PastEventsWeeks: 2,
		},
		Output: OutputConfig{
			HTMLPath: "public/index.html",
			JSONPath: "public/events.json",
		},
		Snapshot: SnapshotConfig{
			DataDir: "data",
		},
		Server: ServerConfig{
			Port: 8080,
		},
		Weather: WeatherConfig{
			APIKeyEnv:        "AEMET_API_KEY",
			MunicipalityCode: "28079",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestValidate_MissingCulturalEventsJSON(t *testing.T) {
	cfg := &Config{
		CulturalEvents: CulturalEventsConfig{
			JSONURL: "", // Missing
			XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
			CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
		},
		CityEvents: CityEventsConfig{
			XMLURL: "https://www.esmadrid.com/opendata/agenda_v1_es.xml",
		},
		Filter: FilterConfig{
			Latitude:        40.42338,
			Longitude:       -3.71217,
			RadiusKm:        0.35,
			PastEventsWeeks: 2,
		},
		Output: OutputConfig{
			HTMLPath: "public/index.html",
			JSONPath: "public/events.json",
		},
		Snapshot: SnapshotConfig{
			DataDir: "data",
		},
		Server: ServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded, want error for missing cultural_events.json_url")
	}
	if !strings.Contains(err.Error(), "cultural_events.json_url") {
		t.Errorf("Validate() error = %v, want error containing 'cultural_events.json_url'", err)
	}
}

func TestValidate_MissingCityEventsXML(t *testing.T) {
	cfg := &Config{
		CulturalEvents: CulturalEventsConfig{
			JSONURL: "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
			XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
			CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
		},
		CityEvents: CityEventsConfig{
			XMLURL: "", // Missing
		},
		Filter: FilterConfig{
			Latitude:        40.42338,
			Longitude:       -3.71217,
			RadiusKm:        0.35,
			PastEventsWeeks: 2,
		},
		Output: OutputConfig{
			HTMLPath: "public/index.html",
			JSONPath: "public/events.json",
		},
		Snapshot: SnapshotConfig{
			DataDir: "data",
		},
		Server: ServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded, want error for missing city_events.xml_url")
	}
	if !strings.Contains(err.Error(), "city_events.xml_url") {
		t.Errorf("Validate() error = %v, want error containing 'city_events.xml_url'", err)
	}
}

func TestValidate_InvalidCoordinates(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
		want string
	}{
		{
			name: "latitude too low",
			lat:  -91.0,
			lon:  -3.71217,
			want: "latitude",
		},
		{
			name: "latitude too high",
			lat:  91.0,
			lon:  -3.71217,
			want: "latitude",
		},
		{
			name: "longitude too low",
			lat:  40.42338,
			lon:  -181.0,
			want: "longitude",
		},
		{
			name: "longitude too high",
			lat:  40.42338,
			lon:  181.0,
			want: "longitude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				CulturalEvents: CulturalEventsConfig{
					JSONURL: "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
					XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
					CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
				},
				CityEvents: CityEventsConfig{
					XMLURL: "https://www.esmadrid.com/opendata/agenda_v1_es.xml",
				},
				Filter: FilterConfig{
					Latitude:        tt.lat,
					Longitude:       tt.lon,
					RadiusKm:        0.35,
					PastEventsWeeks: 2,
				},
				Output: OutputConfig{
					HTMLPath: "public/index.html",
					JSONPath: "public/events.json",
				},
				Snapshot: SnapshotConfig{
					DataDir: "data",
				},
				Server: ServerConfig{
					Port: 8080,
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() succeeded, want error for invalid %s", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestValidate_InvalidRadiusKm(t *testing.T) {
	cfg := &Config{
		CulturalEvents: CulturalEventsConfig{
			JSONURL: "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
			XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
			CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
		},
		CityEvents: CityEventsConfig{
			XMLURL: "https://www.esmadrid.com/opendata/agenda_v1_es.xml",
		},
		Filter: FilterConfig{
			Latitude:        40.42338,
			Longitude:       -3.71217,
			RadiusKm:        -0.5, // Invalid
			PastEventsWeeks: 2,
		},
		Output: OutputConfig{
			HTMLPath: "public/index.html",
			JSONPath: "public/events.json",
		},
		Snapshot: SnapshotConfig{
			DataDir: "data",
		},
		Server: ServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded, want error for invalid radius_km")
	}
	if !strings.Contains(err.Error(), "radius_km") {
		t.Errorf("Validate() error = %v, want error containing 'radius_km'", err)
	}
}

func TestValidate_MissingOutputPaths(t *testing.T) {
	tests := []struct {
		name     string
		htmlPath string
		jsonPath string
		want     string
	}{
		{
			name:     "missing html_path",
			htmlPath: "",
			jsonPath: "public/events.json",
			want:     "html_path",
		},
		{
			name:     "missing json_path",
			htmlPath: "public/index.html",
			jsonPath: "",
			want:     "json_path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				CulturalEvents: CulturalEventsConfig{
					JSONURL: "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
					XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
					CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
				},
				CityEvents: CityEventsConfig{
					XMLURL: "https://www.esmadrid.com/opendata/agenda_v1_es.xml",
				},
				Filter: FilterConfig{
					Latitude:        40.42338,
					Longitude:       -3.71217,
					RadiusKm:        0.35,
					PastEventsWeeks: 2,
				},
				Output: OutputConfig{
					HTMLPath: tt.htmlPath,
					JSONPath: tt.jsonPath,
				},
				Snapshot: SnapshotConfig{
					DataDir: "data",
				},
				Server: ServerConfig{
					Port: 8080,
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() succeeded, want error for missing %s", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestValidate_MissingDataDir(t *testing.T) {
	cfg := &Config{
		CulturalEvents: CulturalEventsConfig{
			JSONURL: "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
			XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
			CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
		},
		CityEvents: CityEventsConfig{
			XMLURL: "https://www.esmadrid.com/opendata/agenda_v1_es.xml",
		},
		Filter: FilterConfig{
			Latitude:        40.42338,
			Longitude:       -3.71217,
			RadiusKm:        0.35,
			PastEventsWeeks: 2,
		},
		Output: OutputConfig{
			HTMLPath: "public/index.html",
			JSONPath: "public/events.json",
		},
		Snapshot: SnapshotConfig{
			DataDir: "", // Missing
		},
		Server: ServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded, want error for missing data_dir")
	}
	if !strings.Contains(err.Error(), "data_dir") {
		t.Errorf("Validate() error = %v, want error containing 'data_dir'", err)
	}
}

func TestValidate_InvalidServerPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{
			name: "port zero",
			port: 0,
		},
		{
			name: "port negative",
			port: -1,
		},
		{
			name: "port too high",
			port: 65536,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				CulturalEvents: CulturalEventsConfig{
					JSONURL: "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
					XMLURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml",
					CSVURL:  "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv",
				},
				CityEvents: CityEventsConfig{
					XMLURL: "https://www.esmadrid.com/opendata/agenda_v1_es.xml",
				},
				Filter: FilterConfig{
					Latitude:        40.42338,
					Longitude:       -3.71217,
					RadiusKm:        0.35,
					PastEventsWeeks: 2,
				},
				Output: OutputConfig{
					HTMLPath: "public/index.html",
					JSONPath: "public/events.json",
				},
				Snapshot: SnapshotConfig{
					DataDir: "data",
				},
				Server: ServerConfig{
					Port: tt.port,
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() succeeded, want error for invalid port %d", tt.port)
			}
			if !strings.Contains(err.Error(), "port") {
				t.Errorf("Validate() error = %v, want error containing 'port'", err)
			}
		})
	}
}
