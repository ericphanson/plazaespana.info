package report

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestBuildReport_WriteText(t *testing.T) {
	// Create a sample report
	report := &BuildReport{
		BuildTime:   time.Date(2025, 10, 20, 10, 30, 0, 0, time.UTC),
		Duration:    2 * time.Second,
		ExitStatus:  "SUCCESS",
		EventsCount: 42,
	}

	report.Fetching = FetchReport{
		Attempts: []FetchAttempt{
			{
				Source:      "JSON",
				URL:         "https://example.com/events.json",
				Status:      "FAILED",
				Duration:    500 * time.Millisecond,
				Error:       "invalid character",
				HTTPStatus:  200,
				ContentType: "application/json",
				Size:        1024000,
			},
			{
				Source:      "CSV",
				URL:         "https://example.com/events.csv",
				Status:      "SUCCESS",
				Duration:    300 * time.Millisecond,
				HTTPStatus:  200,
				ContentType: "text/csv",
				Size:        850000,
				EventCount:  1001,
			},
		},
		SourceUsed:    "CSV",
		TotalDuration: 800 * time.Millisecond,
	}

	report.Processing = ProcessingReport{
		Deduplication: DeduplicationStats{
			Input:      1001,
			Duplicates: 0,
			Output:     1001,
			Duration:   10 * time.Millisecond,
		},
		GeoFilter: GeoFilterStats{
			RefLat:        40.42338,
			RefLon:        -3.71217,
			Radius:        2.0,
			Input:         1001,
			MissingCoords: 94,
			OutsideRadius: 865,
			Kept:          42,
			Duration:      50 * time.Millisecond,
		},
		TimeFilter: TimeFilterStats{
			ReferenceTime: time.Date(2025, 10, 20, 10, 30, 0, 0, time.UTC),
			Timezone:      "Europe/Madrid",
			Input:         42,
			ParseFailures: 0,
			PastEvents:    0,
			Kept:          42,
			Duration:      20 * time.Millisecond,
		},
	}

	report.Output = OutputReport{
		HTML: OutputFile{
			Path:     "./public/index.html",
			Size:     8192,
			Status:   "SUCCESS",
			Duration: 10 * time.Millisecond,
		},
		JSON: OutputFile{
			Path:     "./public/events.json",
			Size:     4096,
			Status:   "SUCCESS",
			Duration: 5 * time.Millisecond,
		},
	}

	report.AddWarning("Geographic radius restrictive - only %.1f%% kept", 4.2)
	report.AddRecommendation("Consider increasing radius to 2-3km")

	// Write report
	var buf bytes.Buffer
	err := report.WriteText(&buf)

	// Verify
	if err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()

	// Check structure
	requiredSections := []string{
		"MADRID EVENTS SITE BUILD REPORT",
		"1. DATA FETCHING",
		"2. DATA PROCESSING",
		"4. OUTPUT GENERATION",
		"5. SUMMARY",
	}

	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("Report missing section: %s", section)
		}
	}

	// Check specific content
	if !strings.Contains(output, "Exit Status: SUCCESS") {
		t.Error("Missing exit status")
	}

	if !strings.Contains(output, "Events Generated: 42") {
		t.Error("Missing event count")
	}

	if !strings.Contains(output, "Attempt 1: JSON Source") {
		t.Error("Missing JSON attempt")
	}

	if !strings.Contains(output, "Status: FAILED") {
		t.Error("Missing failure status")
	}

	if !strings.Contains(output, "Data Source Used: CSV") {
		t.Error("Missing source used")
	}

	if !strings.Contains(output, "Warnings: 1") {
		t.Error("Missing warnings section")
	}

	if !strings.Contains(output, "Recommendations:") {
		t.Error("Missing recommendations")
	}
}

func TestNewBuildReport(t *testing.T) {
	report := NewBuildReport()

	if report == nil {
		t.Fatal("NewBuildReport returned nil")
	}

	if report.ExitStatus != "SUCCESS" {
		t.Errorf("Expected initial status SUCCESS, got %s", report.ExitStatus)
	}

	if report.BuildTime.IsZero() {
		t.Error("BuildTime not set")
	}
}

func TestAddDataQualityIssue(t *testing.T) {
	report := NewBuildReport()

	issue := DataQualityIssue{
		Type:        "ENCODING",
		Severity:    "WARNING",
		Count:       15,
		Description: "UTF-8 encoding issues detected",
		Examples:    []string{"Madrid Art D�co"},
	}

	report.AddDataQualityIssue(issue)

	if len(report.DataQuality) != 1 {
		t.Errorf("Expected 1 data quality issue, got %d", len(report.DataQuality))
	}

	if report.DataQuality[0].Type != "ENCODING" {
		t.Error("Data quality issue not added correctly")
	}
}
