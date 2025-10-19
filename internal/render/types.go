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
}
