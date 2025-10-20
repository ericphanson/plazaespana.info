package fetch

import "time"

// ClientMode controls fetch behavior for different environments.
type ClientMode string

const (
	// ProductionMode: Normal operation (hourly cron)
	// - Respectful delays between requests
	// - HTTP caching with If-Modified-Since
	// - Cache TTL: 30 minutes (cache expires if data < 30 min old)
	ProductionMode ClientMode = "production"

	// DevelopmentMode: Testing/debugging (frequent builds)
	// - Aggressive local caching (cache TTL: 1 hour)
	// - WARNING printed to console if making real request
	// - Max 1 request per URL per 5 minutes
	DevelopmentMode ClientMode = "development"
)

// ModeConfig holds mode-specific configuration.
type ModeConfig struct {
	Mode           ClientMode
	CacheTTL       time.Duration // How long to trust cached data
	MinDelay       time.Duration // Minimum delay between requests to same host
	MaxRequestRate int           // Max requests per URL per time window
	TimeWindow     time.Duration // Time window for rate limiting
}

// DefaultProductionConfig returns production mode configuration.
func DefaultProductionConfig() ModeConfig {
	return ModeConfig{
		Mode:           ProductionMode,
		CacheTTL:       30 * time.Minute,
		MinDelay:       2 * time.Second,
		MaxRequestRate: 1, // 1 request per time window
		TimeWindow:     1 * time.Hour,
	}
}

// DefaultDevelopmentConfig returns development mode configuration.
func DefaultDevelopmentConfig() ModeConfig {
	return ModeConfig{
		Mode:           DevelopmentMode,
		CacheTTL:       1 * time.Hour,
		MinDelay:       5 * time.Second,
		MaxRequestRate: 1,             // 1 request per 5 minutes
		TimeWindow:     5 * time.Minute,
	}
}

// ParseMode converts a string to ClientMode.
func ParseMode(s string) ClientMode {
	switch s {
	case "production", "prod":
		return ProductionMode
	case "development", "dev":
		return DevelopmentMode
	default:
		return ProductionMode // Safe default
	}
}
