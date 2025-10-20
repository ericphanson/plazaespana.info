package report

import (
	"fmt"
	"time"
)

// BuildReport tracks the entire build process.
type BuildReport struct {
	BuildTime   time.Time
	Duration    time.Duration
	ExitStatus  string // "SUCCESS", "FAILED", "PARTIAL"
	EventsCount int

	Fetching    FetchReport
	Processing  ProcessingReport
	DataQuality []DataQualityIssue
	Output      OutputReport

	Warnings        []string
	Recommendations []string
}

// FetchReport tracks data fetching attempts.
type FetchReport struct {
	JSON FetchAttempt
	XML  FetchAttempt
	CSV  FetchAttempt

	TotalDuration time.Duration
}

// FetchAttempt represents one attempt to fetch data.
type FetchAttempt struct {
	Source      string // "JSON", "XML", "CSV"
	URL         string
	Status      string // "SUCCESS", "FAILED", "SKIPPED"
	Duration    time.Duration
	Error       string
	HTTPStatus  int
	ContentType string
	Size        int64
	EventCount  int
}

// ProcessingReport tracks data processing steps.
type ProcessingReport struct {
	Merge      MergeStats
	GeoFilter  GeoFilterStats
	TimeFilter TimeFilterStats
}

// MergeStats tracks multi-source merging and deduplication.
type MergeStats struct {
	JSONEvents int
	XMLEvents  int
	CSVEvents  int

	TotalBeforeMerge int
	UniqueEvents     int
	Duplicates       int

	// Source coverage
	InAllThree   int // Events found in all 3 sources
	InTwoSources int // Events found in 2 sources
	InOneSource  int // Events found in only 1 source

	Duration time.Duration
}

// DeduplicationStats tracks event deduplication.
type DeduplicationStats struct {
	Input      int
	Duplicates int
	Output     int
	Duration   time.Duration
}

// GeoFilterStats tracks geographic filtering.
type GeoFilterStats struct {
	RefLat        float64
	RefLon        float64
	Radius        float64
	Input         int
	MissingCoords int
	OutsideRadius int
	Kept          int
	Duration      time.Duration
}

// TimeFilterStats tracks time-based filtering.
type TimeFilterStats struct {
	ReferenceTime time.Time
	Timezone      string
	Input         int
	ParseFailures int
	PastEvents    int
	Kept          int
	Duration      time.Duration
}

// DataQualityIssue represents a data quality problem.
type DataQualityIssue struct {
	Type           string   // "ENCODING", "MISSING_FIELD", "INVALID_FORMAT", etc.
	Severity       string   // "INFO", "WARNING", "ERROR"
	Count          int      // Number of occurrences
	Description    string   // Human-readable description
	Examples       []string // Sample problematic values
	Recommendation string   // Suggested fix
}

// OutputReport tracks output file generation.
type OutputReport struct {
	HTML     OutputFile
	JSON     OutputFile
	Snapshot OutputFile
}

// OutputFile represents a generated output file.
type OutputFile struct {
	Path     string
	Size     int64
	Status   string // "SUCCESS", "FAILED", "SKIPPED"
	Error    string
	Duration time.Duration
}

// NewBuildReport creates a new report initialized with defaults.
func NewBuildReport() *BuildReport {
	return &BuildReport{
		BuildTime:  time.Now(),
		ExitStatus: "SUCCESS",
	}
}

// AddWarning adds a warning message to the report.
func (r *BuildReport) AddWarning(format string, args ...interface{}) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// AddRecommendation adds a recommendation to the report.
func (r *BuildReport) AddRecommendation(format string, args ...interface{}) {
	r.Recommendations = append(r.Recommendations, fmt.Sprintf(format, args...))
}

// AddDataQualityIssue adds a data quality issue to the report.
func (r *BuildReport) AddDataQualityIssue(issue DataQualityIssue) {
	r.DataQuality = append(r.DataQuality, issue)
}
