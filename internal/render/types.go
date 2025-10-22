package render

import "time"

// TemplateData holds data for HTML template rendering.
type TemplateData struct {
	Lang           string
	CSSHash        string
	LastUpdated    string
	CulturalEvents []TemplateEvent
	CityEvents     []TemplateEvent
	TotalEvents    int
}

// TemplateEvent represents an event for template rendering.
type TemplateEvent struct {
	IDEvento          string
	Titulo            string
	StartHuman        string
	StartTime         time.Time // For sorting
	NombreInstalacion string
	ContentURL        string
	Description       string // Truncated description
	EventType         string // "city" or "cultural"
}

// JSONEvent represents an event in the machine-readable JSON output.
type JSONEvent struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time,omitempty"`
	VenueName  string `json:"venue_name,omitempty"`
	DetailsURL string `json:"details_url,omitempty"`
}

// JSONOutput is the top-level structure for the JSON API output.
type JSONOutput struct {
	CulturalEvents []JSONEvent `json:"cultural_events"`
	CityEvents     []JSONEvent `json:"city_events"`
	Meta           JSONMeta    `json:"meta"`
}

// JSONMeta contains metadata about the JSON output.
type JSONMeta struct {
	UpdateTime    string `json:"update_time"`
	TotalCultural int    `json:"total_cultural"`
	TotalCity     int    `json:"total_city"`
}
