package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the complete application configuration.
type Config struct {
	CulturalEvents CulturalEventsConfig `toml:"cultural_events"`
	CityEvents     CityEventsConfig     `toml:"city_events"`
	Filter         FilterConfig         `toml:"filter"`
	Output         OutputConfig         `toml:"output"`
	Snapshot       SnapshotConfig       `toml:"snapshot"`
	Server         ServerConfig         `toml:"server"`
}

// CulturalEventsConfig holds configuration for datos.madrid.es cultural programming.
type CulturalEventsConfig struct {
	JSONURL string `toml:"json_url"`
	XMLURL  string `toml:"xml_url"`
	CSVURL  string `toml:"csv_url"`
}

// CityEventsConfig holds configuration for esmadrid.com tourism/city events.
type CityEventsConfig struct {
	XMLURL string `toml:"xml_url"`
}

// FilterConfig holds event filtering criteria.
type FilterConfig struct {
	Latitude        float64  `toml:"latitude"`
	Longitude       float64  `toml:"longitude"`
	RadiusKm        float64  `toml:"radius_km"`
	Distritos       []string `toml:"distritos"`
	PastEventsWeeks int      `toml:"past_events_weeks"`
}

// OutputConfig holds output file paths.
type OutputConfig struct {
	HTMLPath string `toml:"html_path"`
	JSONPath string `toml:"json_path"`
}

// SnapshotConfig holds snapshot directory configuration.
type SnapshotConfig struct {
	DataDir string `toml:"data_dir"`
}

// ServerConfig holds development server settings.
type ServerConfig struct {
	Port int `toml:"port"`
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{
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
			Distritos:       []string{"CENTRO", "MONCLOA-ARAVACA"},
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
}

// Load reads and parses a TOML configuration file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks that all required configuration fields are set correctly.
func (c *Config) Validate() error {
	// Validate CulturalEvents URLs
	if c.CulturalEvents.JSONURL == "" {
		return fmt.Errorf("cultural_events.json_url must not be empty")
	}
	if c.CulturalEvents.XMLURL == "" {
		return fmt.Errorf("cultural_events.xml_url must not be empty")
	}
	if c.CulturalEvents.CSVURL == "" {
		return fmt.Errorf("cultural_events.csv_url must not be empty")
	}

	// Validate CityEvents URLs
	if c.CityEvents.XMLURL == "" {
		return fmt.Errorf("city_events.xml_url must not be empty")
	}

	// Validate coordinates
	if c.Filter.Latitude < -90 || c.Filter.Latitude > 90 {
		return fmt.Errorf("filter.latitude must be between -90 and 90, got %f", c.Filter.Latitude)
	}
	if c.Filter.Longitude < -180 || c.Filter.Longitude > 180 {
		return fmt.Errorf("filter.longitude must be between -180 and 180, got %f", c.Filter.Longitude)
	}

	// Validate radius
	if c.Filter.RadiusKm <= 0 {
		return fmt.Errorf("filter.radius_km must be positive, got %f", c.Filter.RadiusKm)
	}

	// Validate output paths
	if c.Output.HTMLPath == "" {
		return fmt.Errorf("output.html_path must not be empty")
	}
	if c.Output.JSONPath == "" {
		return fmt.Errorf("output.json_path must not be empty")
	}

	// Validate snapshot data directory
	if c.Snapshot.DataDir == "" {
		return fmt.Errorf("snapshot.data_dir must not be empty")
	}

	// Validate server port
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}

	return nil
}
