package render

// TemplateData holds data for HTML template rendering.
type TemplateData struct {
	Lang        string
	CSSHash     string
	LastUpdated string
	Events      []TemplateEvent
}

// TemplateEvent represents an event for template rendering.
type TemplateEvent struct {
	IDEvento          string
	Titulo            string
	StartHuman        string
	NombreInstalacion string
	ContentURL        string
	Description       string // Truncated description
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
