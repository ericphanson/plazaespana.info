package audit

// AuditParseError represents a parse error in the audit file.
// Serializable version of event.ParseError with JSON-friendly error field.
type AuditParseError struct {
	Source      string `json:"source"`       // "JSON", "XML", "CSV", "ESMadrid"
	Index       int    `json:"index"`        // Row/record index in source
	ID          string `json:"id,omitempty"` // Event ID if available
	RawData     string `json:"raw_data,omitempty"`
	Error       string `json:"error"`        // Error message (string, not error type)
	RecoverType string `json:"recover_type"` // "skipped", "partial", "defaulted"
}

// ParseErrorsAudit tracks parse errors from both pipelines.
type ParseErrorsAudit struct {
	CulturalErrors []AuditParseError `json:"cultural"`
	CityErrors     []AuditParseError `json:"city"`
	TotalErrors    int               `json:"total_errors"`
}
