package report

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// BuildReport contains comprehensive information about a site build.
type BuildReport struct {
	BuildTime   time.Time
	Duration    time.Duration
	ExitStatus  string // "SUCCESS", "PARTIAL", "FAILED"
	EventsCount int

	Fetching    FetchReport
	Processing  ProcessingReport
	DataQuality []DataQualityIssue
	Output      OutputReport

	Warnings        []string
	Recommendations []string
}

// FetchReport tracks all data fetching attempts.
type FetchReport struct {
	Attempts      []FetchAttempt
	SourceUsed    string // "JSON", "XML", "CSV", "SNAPSHOT"
	TotalDuration time.Duration
}

// FetchAttempt represents a single attempt to fetch data from a source.
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
	Deduplication DeduplicationStats
	GeoFilter     GeoFilterStats
	TimeFilter    TimeFilterStats
}

// DeduplicationStats tracks deduplication results.
type DeduplicationStats struct {
	Input      int
	Duplicates int
	Output     int
	Duration   time.Duration
}

// GeoFilterStats tracks geographic filtering results.
type GeoFilterStats struct {
	RefLat float64
	RefLon float64
	Radius float64

	Input         int
	MissingCoords int
	OutsideRadius int
	Kept          int

	Duration time.Duration
}

// TimeFilterStats tracks time-based filtering results.
type TimeFilterStats struct {
	ReferenceTime time.Time
	Timezone      string

	Input         int
	ParseFailures int
	PastEvents    int
	Kept          int

	Duration time.Duration
}

// DataQualityIssue represents a data quality problem.
type DataQualityIssue struct {
	Type           string // "ENCODING", "MISSING_DATA", "PARSE_ERROR"
	Severity       string // "WARNING", "ERROR", "INFO"
	Count          int
	Description    string
	Examples       []string // Up to 3 examples
	Recommendation string
}

// OutputReport tracks output file generation.
type OutputReport struct {
	HTML     OutputFile
	JSON     OutputFile
	Snapshot OutputFile
}

// OutputFile represents information about a generated file.
type OutputFile struct {
	Path     string
	Size     int64
	Status   string // "SUCCESS", "FAILED", "SKIPPED"
	Duration time.Duration
	Error    string
}

// NewBuildReport creates a new build report initialized with current time.
func NewBuildReport() *BuildReport {
	return &BuildReport{
		BuildTime:  time.Now(),
		ExitStatus: "SUCCESS", // Assume success, set to FAILED if issues
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

// WriteText writes a human-readable text report.
func (r *BuildReport) WriteText(w io.Writer) error {
	var b strings.Builder

	// Header
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("MADRID EVENTS SITE BUILD REPORT\n")
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString(fmt.Sprintf("Build Time: %s\n", r.BuildTime.Format("2006-01-02 15:04:05 MST")))
	b.WriteString(fmt.Sprintf("Duration: %.2fs\n", r.Duration.Seconds()))
	b.WriteString(fmt.Sprintf("Exit Status: %s\n", r.ExitStatus))
	b.WriteString(fmt.Sprintf("Events Generated: %d\n", r.EventsCount))
	b.WriteString("\n")

	// Data Fetching
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("1. DATA FETCHING\n")
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("\n")

	for i, attempt := range r.Fetching.Attempts {
		b.WriteString(fmt.Sprintf("Attempt %d: %s Source\n", i+1, attempt.Source))
		b.WriteString(fmt.Sprintf("  URL: %s\n", attempt.URL))
		b.WriteString(fmt.Sprintf("  Status: %s\n", attempt.Status))
		b.WriteString(fmt.Sprintf("  Duration: %.2fs\n", attempt.Duration.Seconds()))

		if attempt.Status == "FAILED" {
			b.WriteString(fmt.Sprintf("  Error: %s\n", attempt.Error))
		} else if attempt.Status == "SUCCESS" {
			b.WriteString(fmt.Sprintf("  HTTP Status: %d\n", attempt.HTTPStatus))
			b.WriteString(fmt.Sprintf("  Content-Type: %s\n", attempt.ContentType))
			b.WriteString(fmt.Sprintf("  Response Size: %.2f KB\n", float64(attempt.Size)/1024))
			b.WriteString(fmt.Sprintf("  Events Parsed: %d\n", attempt.EventCount))
		}

		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Data Source Used: %s\n", r.Fetching.SourceUsed))
	b.WriteString(fmt.Sprintf("Total Fetch Duration: %.2fs\n", r.Fetching.TotalDuration.Seconds()))
	b.WriteString("\n")

	// Data Processing
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("2. DATA PROCESSING\n")
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("\n")

	// Deduplication
	b.WriteString("Deduplication (by ID-EVENTO):\n")
	b.WriteString(fmt.Sprintf("  Input: %d events\n", r.Processing.Deduplication.Input))
	b.WriteString(fmt.Sprintf("  Duplicates removed: %d\n", r.Processing.Deduplication.Duplicates))
	b.WriteString(fmt.Sprintf("  Output: %d events\n", r.Processing.Deduplication.Output))
	b.WriteString(fmt.Sprintf("  Duration: %.3fs\n", r.Processing.Deduplication.Duration.Seconds()))
	b.WriteString("\n")

	// Geographic filtering
	b.WriteString("Geographic Filtering (Haversine):\n")
	b.WriteString(fmt.Sprintf("  Reference Point: (%.5f, %.5f)\n", r.Processing.GeoFilter.RefLat, r.Processing.GeoFilter.RefLon))
	b.WriteString(fmt.Sprintf("  Radius: %.2f km\n", r.Processing.GeoFilter.Radius))
	b.WriteString("\n")
	b.WriteString("  Filtered Out:\n")
	b.WriteString(fmt.Sprintf("    - Missing coordinates: %d events (%.1f%%)\n",
		r.Processing.GeoFilter.MissingCoords,
		percent(r.Processing.GeoFilter.MissingCoords, r.Processing.GeoFilter.Input)))
	b.WriteString(fmt.Sprintf("    - Outside radius: %d events (%.1f%%)\n",
		r.Processing.GeoFilter.OutsideRadius,
		percent(r.Processing.GeoFilter.OutsideRadius, r.Processing.GeoFilter.Input)))
	b.WriteString(fmt.Sprintf("  Kept: %d events (%.1f%%)\n",
		r.Processing.GeoFilter.Kept,
		percent(r.Processing.GeoFilter.Kept, r.Processing.GeoFilter.Input)))
	b.WriteString(fmt.Sprintf("  Duration: %.3fs\n", r.Processing.GeoFilter.Duration.Seconds()))
	b.WriteString("\n")

	// Time filtering
	b.WriteString("Time Filtering (Future Events):\n")
	b.WriteString(fmt.Sprintf("  Reference Time: %s\n", r.Processing.TimeFilter.ReferenceTime.Format("2006-01-02 15:04:05 MST")))
	b.WriteString(fmt.Sprintf("  Timezone: %s\n", r.Processing.TimeFilter.Timezone))
	b.WriteString("\n")
	b.WriteString("  Filtered Out:\n")
	b.WriteString(fmt.Sprintf("    - Parse failures: %d events (%.1f%%)\n",
		r.Processing.TimeFilter.ParseFailures,
		percent(r.Processing.TimeFilter.ParseFailures, r.Processing.TimeFilter.Input)))
	b.WriteString(fmt.Sprintf("    - Past events: %d events (%.1f%%)\n",
		r.Processing.TimeFilter.PastEvents,
		percent(r.Processing.TimeFilter.PastEvents, r.Processing.TimeFilter.Input)))
	b.WriteString(fmt.Sprintf("  Kept: %d events (%.1f%%)\n",
		r.Processing.TimeFilter.Kept,
		percent(r.Processing.TimeFilter.Kept, r.Processing.TimeFilter.Input)))
	b.WriteString(fmt.Sprintf("  Duration: %.3fs\n", r.Processing.TimeFilter.Duration.Seconds()))
	b.WriteString("\n")

	// Data Quality Issues
	if len(r.DataQuality) > 0 {
		b.WriteString(strings.Repeat("=", 80) + "\n")
		b.WriteString("3. DATA QUALITY ISSUES\n")
		b.WriteString(strings.Repeat("=", 80) + "\n")
		b.WriteString("\n")

		for _, issue := range r.DataQuality {
			symbol := "ℹ"
			if issue.Severity == "WARNING" {
				symbol = "⚠"
			} else if issue.Severity == "ERROR" {
				symbol = "✗"
			}

			b.WriteString(fmt.Sprintf("%s %s: %d occurrences\n", symbol, issue.Type, issue.Count))
			b.WriteString(fmt.Sprintf("  %s\n", issue.Description))

			if len(issue.Examples) > 0 {
				b.WriteString("  Examples:\n")
				for _, example := range issue.Examples {
					b.WriteString(fmt.Sprintf("    - %s\n", example))
				}
			}

			if issue.Recommendation != "" {
				b.WriteString(fmt.Sprintf("  Recommendation: %s\n", issue.Recommendation))
			}

			b.WriteString("\n")
		}
	}

	// Output Generation
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("4. OUTPUT GENERATION\n")
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("\n")

	writeOutputFile := func(name string, file OutputFile) {
		b.WriteString(fmt.Sprintf("%s:\n", name))
		b.WriteString(fmt.Sprintf("  Path: %s\n", file.Path))
		b.WriteString(fmt.Sprintf("  Status: %s\n", file.Status))
		if file.Status == "SUCCESS" {
			b.WriteString(fmt.Sprintf("  File size: %.2f KB\n", float64(file.Size)/1024))
		} else if file.Status == "FAILED" {
			b.WriteString(fmt.Sprintf("  Error: %s\n", file.Error))
		}
		b.WriteString(fmt.Sprintf("  Duration: %.3fs\n", file.Duration.Seconds()))
		b.WriteString("\n")
	}

	writeOutputFile("HTML Generation", r.Output.HTML)
	writeOutputFile("JSON API Generation", r.Output.JSON)
	writeOutputFile("Snapshot Saved", r.Output.Snapshot)

	// Summary
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("5. SUMMARY\n")
	b.WriteString(strings.Repeat("=", 80) + "\n")
	b.WriteString("\n")

	statusSymbol := "✓"
	if r.ExitStatus == "FAILED" {
		statusSymbol = "✗"
	} else if r.ExitStatus == "PARTIAL" {
		statusSymbol = "⚠"
	}
	b.WriteString(fmt.Sprintf("Build Status: %s %s\n\n", statusSymbol, r.ExitStatus))

	b.WriteString("Events Pipeline:\n")
	b.WriteString(fmt.Sprintf("  Fetched: %d (from %s)\n", r.Processing.Deduplication.Input, r.Fetching.SourceUsed))
	b.WriteString(fmt.Sprintf("  After dedup: %d\n", r.Processing.Deduplication.Output))
	b.WriteString(fmt.Sprintf("  After geo filter: %d (%.2fkm radius)\n", r.Processing.GeoFilter.Kept, r.Processing.GeoFilter.Radius))
	b.WriteString(fmt.Sprintf("  After time filter: %d\n", r.Processing.TimeFilter.Kept))
	b.WriteString(fmt.Sprintf("  Final output: %d\n", r.EventsCount))
	b.WriteString("\n")

	// Warnings
	if len(r.Warnings) > 0 {
		b.WriteString(fmt.Sprintf("Warnings: %d\n", len(r.Warnings)))
		for i, warning := range r.Warnings {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, warning))
		}
		b.WriteString("\n")
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		b.WriteString("Recommendations:\n")
		for i, rec := range r.Recommendations {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, rec))
		}
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("=", 80) + "\n")

	_, err := w.Write([]byte(b.String()))
	return err
}

// percent calculates percentage, handling division by zero.
func percent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) * 100.0 / float64(total)
}
