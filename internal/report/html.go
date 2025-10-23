package report

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// SVG icon constants for build report (Bootstrap Icons)
const (
	iconTheater   = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 16 16" fill="#7c3aed" style="vertical-align: middle; margin-right: 0.25em;"><path d="M0 4a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H1a1 1 0 0 1-1-1V4zm3.5 5.5a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1zm9 0a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1zM5 6a1 1 0 1 0 0 2 1 1 0 0 0 0-2zm6 0a1 1 0 1 0 0 2 1 1 0 0 0 0-2zM8 6c-.646 0-1.278.285-1.67.765a.5.5 0 0 0 .74.673A1.238 1.238 0 0 1 8 7c.345 0 .678.143.93.438a.5.5 0 0 0 .74-.673A2.238 2.238 0 0 0 8 6z"/></svg>`
	iconConfetti  = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 16 16" fill="#ea580c" style="vertical-align: middle; margin-right: 0.25em;"><path d="M4 8a.5.5 0 0 1 .5-.5h7a.5.5 0 0 1 0 1h-7A.5.5 0 0 1 4 8z"/><path d="M1 0 0 1l2.2 3.081a1 1 0 0 0 .815.419h.07a1 1 0 0 1 .708.293l2.675 2.675-2.617 2.654A3.003 3.003 0 0 0 0 13a3 3 0 1 0 5.878-.851l2.654-2.617.968.968-.305.914a1 1 0 0 0 .242 1.023l3.356 3.356a1 1 0 0 0 1.414 0l1.586-1.586a1 1 0 0 0 0-1.414l-3.356-3.356a1 1 0 0 0-1.023-.242L10.5 9.5l-.96-.96 2.68-2.643A3.005 3.005 0 0 0 16 3c0-.269-.035-.53-.102-.777l-2.14 2.141L12 4l-.364-1.757L13.777.102a3 3 0 0 0-3.675 3.68L7.462 6.46 4.793 3.793a1 1 0 0 1-.293-.707v-.071a1 1 0 0 0-.419-.814L1 0zm9.646 10.646a.5.5 0 0 1 .708 0l3 3a.5.5 0 0 1-.708.708l-3-3a.5.5 0 0 1 0-.708zM3 11l.471.242.529.026.287.445.445.287.026.529L5 13l-.242.471-.026.529-.445.287-.287.445-.529.026L3 15l-.471-.242L2 14.732l-.287-.445L1.268 14l-.026-.529L1 13l.242-.471.026-.529.445-.287.287-.445.529-.026L3 11z"/></svg>`
	iconBroadcast = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M8 16s6-5.686 6-10A6 6 0 0 0 2 6c0 4.314 6 10 6 10zm0-7a3 3 0 1 1 0-6 3 3 0 0 1 0 6z"/></svg>`
	iconSync      = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path fill-rule="evenodd" d="M8 3a5 5 0 1 0 4.546 2.914.5.5 0 0 1 .908-.417A6 6 0 1 1 8 2v1z"/><path d="M8 4.466V.534a.25.25 0 0 1 .41-.192l2.36 1.966c.12.1.12.284 0 .384L8.41 4.658A.25.25 0 0 1 8 4.466z"/></svg>`
	iconMap       = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path fill-rule="evenodd" d="M16 .5a.5.5 0 0 0-.598-.49L10.5.99 5.598.01a.5.5 0 0 0-.196 0l-5 1A.5.5 0 0 0 0 1.5v14a.5.5 0 0 0 .598.49l4.902-.98 4.902.98a.502.502 0 0 0 .196 0l5-1A.5.5 0 0 0 16 14.5V.5zM5 14.09V1.11l.5-.1.5.1v12.98l-.402-.08a.498.498 0 0 0-.196 0L5 14.09zm5 .8V1.91l.402.08a.5.5 0 0 0 .196 0L11 1.91v12.98l-.5.1-.5-.1z"/></svg>`
	iconTarget    = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z"/><path d="M8 13A5 5 0 1 1 8 3a5 5 0 0 1 0 10zm0 1A6 6 0 1 0 8 2a6 6 0 0 0 0 12z"/><path d="M8 11a3 3 0 1 1 0-6 3 3 0 0 1 0 6zm0 1a4 4 0 1 0 0-8 4 4 0 0 0 0 8z"/><path d="M9.5 8a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0z"/></svg>`
	iconClock     = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zM8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71V3.5z"/></svg>`
	iconTag       = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M2 2a1 1 0 0 1 1-1h4.586a1 1 0 0 1 .707.293l7 7a1 1 0 0 1 0 1.414l-4.586 4.586a1 1 0 0 1-1.414 0l-7-7A1 1 0 0 1 2 6.586V2zm3.5 4a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z"/><path d="M1.293 7.793A1 1 0 0 1 1 7.086V2a1 1 0 0 0-1 1v4.586a1 1 0 0 0 .293.707l7 7a1 1 0 0 0 1.414 0l.043-.043-7.457-7.457z"/></svg>`
	iconWarning   = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="#ea580c" style="vertical-align: middle; margin-right: 0.25em;"><path d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"/></svg>`
	iconSuccess   = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="#059669" style="vertical-align: middle; margin-right: 0.15em;"><path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z"/></svg>`
	iconFailed    = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="#dc2626" style="vertical-align: middle; margin-right: 0.15em;"><path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zM5.354 4.646a.5.5 0 1 0-.708.708L7.293 8l-2.647 2.646a.5.5 0 0 0 .708.708L8 8.707l2.646 2.647a.5.5 0 0 0 .708-.708L8.707 8l2.647-2.646a.5.5 0 0 0-.708-.708L8 7.293 5.354 4.646z"/></svg>`
	iconSkipped   = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="#666" style="vertical-align: middle; margin-right: 0.15em;"><path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z"/><path d="M10.97 4.97a.235.235 0 0 0-.02.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-1.071-1.05z"/></svg>`
)

// WriteHTML writes an HTML-formatted build report for dual pipeline architecture.
func (r *BuildReport) WriteHTML(w io.Writer, cssHash string, basePath string) error {
	var b strings.Builder

	// HTML header with external CSS
	b.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Build Report - Madrid Events</title>
  <link rel="stylesheet" href="%s/assets/build-report.%s.css">
</head>
<body>
  <header>
    <h1>Build Report</h1>
    <p class="muted">Madrid Events Site Generator</p>
  </header>

  <main>
`, basePath, cssHash))

	// Build Summary
	b.WriteString(`    <div class="summary-card">
      <h2>Build Summary</h2>
      <div class="summary-grid">
        <div class="summary-item">
          <strong>Build Time</strong>
          <span>` + r.BuildTime.Format("2006-01-02 15:04:05") + `</span>
        </div>
        <div class="summary-item">
          <strong>Duration</strong>
          <span>` + formatDuration(r.Duration) + `</span>
        </div>
        <div class="summary-item">
          <strong>Status</strong>
          <span class="` + statusClass(r.ExitStatus) + `">` + r.ExitStatus + `</span>
        </div>
        <div class="summary-item">
          <strong>Total Events</strong>
          <span>` + fmt.Sprintf("%d", r.TotalEvents) + `</span>
        </div>
      </div>
    </div>
`)

	// Pipeline Overview
	b.WriteString(`    <h2>Pipeline Overview</h2>
    <div class="pipeline-grid">
`)

	// Cultural Pipeline Card
	b.WriteString(fmt.Sprintf(`      <div class="pipeline-card cultural">
        <div class="pipeline-header">
          <span class="icon">%s</span>
          <h3 class="cultural-title">%s</h3>
        </div>
        <div class="pipeline-stat">
          <span>Source</span>
          <span>%s</span>
        </div>
        <div class="pipeline-stat">
          <span>Events</span>
          <span><strong>%d</strong></span>
        </div>
        <div class="pipeline-stat">
          <span>Duration</span>
          <span>%s</span>
        </div>
      </div>
`, iconTheater, r.CulturalPipeline.Name, r.CulturalPipeline.Source, r.CulturalPipeline.EventCount, formatDuration(r.CulturalPipeline.Duration)))

	// City Pipeline Card
	b.WriteString(fmt.Sprintf(`      <div class="pipeline-card city">
        <div class="pipeline-header">
          <span class="icon">%s</span>
          <h3 class="city-title">%s</h3>
        </div>
        <div class="pipeline-stat">
          <span>Source</span>
          <span>%s</span>
        </div>
        <div class="pipeline-stat">
          <span>Events</span>
          <span><strong>%d</strong></span>
        </div>
        <div class="pipeline-stat">
          <span>Duration</span>
          <span>%s</span>
        </div>
      </div>
`, iconConfetti, r.CityPipeline.Name, r.CityPipeline.Source, r.CityPipeline.EventCount, formatDuration(r.CityPipeline.Duration)))

	b.WriteString(`    </div>
`)

	// Cultural Events Pipeline Detailed
	b.WriteString(fmt.Sprintf(`    <h2 class="cultural-title">%s Cultural Events Pipeline</h2>
    <div class="section">
      <h3>%s Data Fetching</h3>
`, iconTheater, iconBroadcast))

	for _, attempt := range r.CulturalPipeline.Fetching.Attempts {
		statusSymbol := iconSuccess
		if attempt.Status == "FAILED" {
			statusSymbol = iconFailed
		} else if attempt.Status == "SKIPPED" {
			statusSymbol = iconSkipped
		}
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>%s%s</span>
        <span>%s</span>
      </div>
`, statusSymbol, attempt.Source, formatAttempt(attempt)))
	}

	if r.CulturalPipeline.Merging != nil {
		b.WriteString(fmt.Sprintf(`      <h3>%s Deduplication</h3>
`, iconSync))
		merge := r.CulturalPipeline.Merging
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Input events</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Duplicates removed</span>
        <span>%d (%.1f%%)</span>
      </div>
      <div class="metric-row">
        <span>Unique events</span>
        <span>%d</span>
      </div>
`, merge.TotalBeforeMerge, merge.Duplicates, float64(merge.Duplicates)*100.0/float64(merge.TotalBeforeMerge), merge.UniqueEvents))
	}

	// Cultural Filtering
	if r.CulturalPipeline.Filtering.DistrictoFilter != nil {
		b.WriteString(fmt.Sprintf(`      <h3>%s Distrito Filtering</h3>
`, iconMap))
		df := r.CulturalPipeline.Filtering.DistrictoFilter
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Allowed districts</span>
        <span>%s</span>
      </div>
      <div class="metric-row">
        <span>Input events</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Kept in district</span>
        <span>%d</span>
      </div>
`, strings.Join(df.AllowedDistricts, ", "), df.Input, df.Kept))
	}

	if r.CulturalPipeline.Filtering.GeoFilter != nil {
		b.WriteString(fmt.Sprintf(`      <h3>%s Geographic Filtering</h3>
`, iconTarget))
		gf := r.CulturalPipeline.Filtering.GeoFilter
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Reference point</span>
        <span>%.5f, %.5f</span>
      </div>
      <div class="metric-row">
        <span>Radius</span>
        <span>%.2f km</span>
      </div>
      <div class="metric-row">
        <span>Input events</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Within radius</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Missing coordinates</span>
        <span>%d (%.1f%%)</span>
      </div>
`, gf.RefLat, gf.RefLon, gf.Radius, gf.Input, gf.Kept, gf.MissingCoords, float64(gf.MissingCoords)*100.0/float64(gf.Input)))
	}

	if r.CulturalPipeline.Filtering.TimeFilter != nil {
		b.WriteString(fmt.Sprintf(`      <h3>%s Time Filtering</h3>
`, iconClock))
		tf := r.CulturalPipeline.Filtering.TimeFilter
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Reference time</span>
        <span>%s</span>
      </div>
      <div class="metric-row">
        <span>Input events</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Past events removed</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Future events kept</span>
        <span>%d</span>
      </div>
`, tf.ReferenceTime.Format("2006-01-02 15:04"), tf.Input, tf.PastEvents, tf.Kept))
	}

	b.WriteString(`    </div>
`)

	// City Events Pipeline Detailed
	b.WriteString(fmt.Sprintf(`    <h2 class="city-title">%s City Events Pipeline</h2>
    <div class="section">
      <h3>%s Data Fetching</h3>
`, iconConfetti, iconBroadcast))

	for _, attempt := range r.CityPipeline.Fetching.Attempts {
		statusSymbol := iconSuccess
		if attempt.Status == "FAILED" {
			statusSymbol = iconFailed
		} else if attempt.Status == "SKIPPED" {
			statusSymbol = iconSkipped
		}
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>%s%s</span>
        <span>%s</span>
      </div>
`, statusSymbol, attempt.Source, formatAttempt(attempt)))
	}

	// City Filtering
	if r.CityPipeline.Filtering.GeoFilter != nil {
		b.WriteString(fmt.Sprintf(`      <h3>%s Geographic Filtering</h3>
`, iconTarget))
		gf := r.CityPipeline.Filtering.GeoFilter
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Reference point</span>
        <span>%.5f, %.5f</span>
      </div>
      <div class="metric-row">
        <span>Radius</span>
        <span>%.2f km</span>
      </div>
      <div class="metric-row">
        <span>Input events</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Within radius</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Missing coordinates</span>
        <span>%d (%.1f%%)</span>
      </div>
`, gf.RefLat, gf.RefLon, gf.Radius, gf.Input, gf.Kept, gf.MissingCoords, float64(gf.MissingCoords)*100.0/float64(gf.Input)))
	}

	if r.CityPipeline.Filtering.CategoryFilter != nil {
		b.WriteString(fmt.Sprintf(`      <h3>%s Category Filtering</h3>
`, iconTag))
		cf := r.CityPipeline.Filtering.CategoryFilter
		if len(cf.AllowedCategories) > 0 {
			b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Allowed categories</span>
        <span>%s</span>
      </div>
`, strings.Join(cf.AllowedCategories, ", ")))
		} else {
			b.WriteString(`      <div class="metric-row">
        <span>Note</span>
        <span>No category filter configured (all kept)</span>
      </div>
`)
		}
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Input events</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Kept</span>
        <span>%d</span>
      </div>
`, cf.Input, cf.Kept))
	}

	if r.CityPipeline.Filtering.TimeFilter != nil {
		b.WriteString(fmt.Sprintf(`      <h3>%s Time Filtering</h3>
`, iconClock))
		tf := r.CityPipeline.Filtering.TimeFilter
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Reference time</span>
        <span>%s</span>
      </div>
      <div class="metric-row">
        <span>Input events</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Past events removed</span>
        <span>%d</span>
      </div>
      <div class="metric-row">
        <span>Future events kept</span>
        <span>%d</span>
      </div>
`, tf.ReferenceTime.Format("2006-01-02 15:04"), tf.Input, tf.PastEvents, tf.Kept))
	}

	b.WriteString(`    </div>
`)

	// Output Files
	b.WriteString(`    <h2>Output Files</h2>
    <div class="section">
`)
	b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>HTML</span>
        <span class="%s">%s</span>
      </div>
      <div class="metric-row">
        <span>JSON</span>
        <span class="%s">%s</span>
      </div>
`, statusClass(r.Output.HTML.Status), r.Output.HTML.Path, statusClass(r.Output.JSON.Status), r.Output.JSON.Path))
	b.WriteString(`    </div>
`)

	// Warnings
	if len(r.Warnings) > 0 {
		b.WriteString(fmt.Sprintf(`    <div class="warning-box">
      <h3>%s Warnings</h3>
      <ul>
`, iconWarning))
		for _, warning := range r.Warnings {
			b.WriteString(fmt.Sprintf("        <li>%s</li>\n", warning))
		}
		b.WriteString(`      </ul>
    </div>
`)
	}

	// Footer
	homeURL := "/"
	if basePath != "" {
		homeURL = basePath + "/"
	}
	b.WriteString(fmt.Sprintf(`  </main>

  <footer>
    <p>Generated by Madrid Events Site Generator</p>
    <p><a href="%s">← Back to events</a></p>
  </footer>
</body>
</html>`, homeURL))

	_, err := w.Write([]byte(b.String()))
	return err
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// statusClass returns the CSS class for a status.
func statusClass(status string) string {
	if status == "SUCCESS" {
		return "status-success"
	}
	return "status-failure"
}

// formatAttempt formats a fetch attempt for display.
func formatAttempt(a FetchAttempt) string {
	if a.Status == "SUCCESS" {
		return fmt.Sprintf("%d events (%s)", a.EventCount, formatDuration(a.Duration))
	}
	if a.Status == "SKIPPED" {
		return "Skipped"
	}
	return fmt.Sprintf("Failed: %s", a.Error)
}
