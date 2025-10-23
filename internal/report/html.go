package report

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// SVG icon constants for build report (Phosphor fill icons)
const (
	iconTheater   = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 256 256" fill="#7c3aed" style="vertical-align: middle; margin-right: 0.25em;"><path d="M216,40H40A16,16,0,0,0,24,56V96a104,104,0,0,0,208,0V56A16,16,0,0,0,216,40ZM96,120a8,8,0,1,1,8-8A8,8,0,0,1,96,120Zm64,0a8,8,0,1,1,8-8A8,8,0,0,1,160,120Zm56-24a88.1,88.1,0,0,1-176,0V56H216Z"/><path d="M128,152a39.94,39.94,0,0,0-33.93,19,8,8,0,0,0,13.86,8,24,24,0,0,1,40.14,0,8,8,0,1,0,13.86-8A39.94,39.94,0,0,0,128,152Z"/></svg>`
	iconConfetti  = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 256 256" fill="#ea580c" style="vertical-align: middle; margin-right: 0.25em;"><path d="M111.49,52.63a15.8,15.8,0,0,0-26,5.77L33,202.78A15.83,15.83,0,0,0,47.76,224a16,16,0,0,0,5.46-1l144.37-52.5a15.8,15.8,0,0,0,5.78-26Zm-8.33,135.21-35-35,13.16-36.21,58.05,58.05Zm-55,20L64,168.1l15.11,15.11ZM192,152.6,103.4,64l27-27.62L192,128Z"/><path d="M144,40a8,8,0,0,1,8-8h16a8,8,0,0,1,0,16H152A8,8,0,0,1,144,40Zm64,72a8,8,0,0,1,8,8v16a8,8,0,0,1-16,0V120A8,8,0,0,1,208,112ZM232,64a8,8,0,0,1-8,8h-8v8a8,8,0,0,1-16,0V72h-8a8,8,0,0,1,0-16h8V48a8,8,0,0,1,16,0v8h8A8,8,0,0,1,232,64ZM184,168h-8a8,8,0,0,0,0,16h8a8,8,0,0,0,0-16Z"/></svg>`
	iconBroadcast = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 256 256" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M128,88a40,40,0,1,0,40,40A40,40,0,0,0,128,88Zm0,64a24,24,0,1,1,24-24A24,24,0,0,1,128,152Zm0-112a88.1,88.1,0,0,0-88,88c0,23.43,13.94,49.52,41.44,77.54A247.16,247.16,0,0,0,122.76,243a8,8,0,0,0,10.48,0,247.16,247.16,0,0,0,41.32-37.46C202.06,177.52,216,151.43,216,128A88.1,88.1,0,0,0,128,40Zm0,206.51C113,233.08,56,176.66,56,128a72,72,0,0,1,144,0C200,176.66,143,233.08,128,246.51Z"/></svg>`
	iconSync      = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 256 256" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M224,48V96a8,8,0,0,1-8,8H168a8,8,0,0,1,0-16H201.42L184.7,71.28a80,80,0,1,0-1.69,114.47,8,8,0,1,1,11.56,11.08A96,96,0,1,1,217.87,68.09L224,62.24V96a8,8,0,0,1-16,0V48A8,8,0,0,1,224,48Z"/></svg>`
	iconMap       = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 256 256" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M228.92,49.69a8,8,0,0,0-6.86-1.45L160.93,63.52,99.58,32.84a8,8,0,0,0-5.52-.6l-64,16A8,8,0,0,0,24,56V200a8,8,0,0,0,9.94,7.76l61.13-15.28,61.35,30.68A8.15,8.15,0,0,0,160,224a8,8,0,0,0,1.94-.24l64-16A8,8,0,0,0,232,200V56A8,8,0,0,0,228.92,49.69ZM104,52.94l48,24V203.06l-48-24ZM40,62.25l48-12v127.5l-48,12Zm176,131.5-48,12V78.25l48-12Z"/></svg>`
	iconTarget    = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 256 256" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M232,128A104,104,0,1,0,79.12,219.82L56.4,242.53a8,8,0,1,0,11.31,11.32l22.61-22.62A104,104,0,0,0,232,128Zm-88-8V88a8,8,0,0,1,16,0v32h32a8,8,0,0,1,0,16H160v32a8,8,0,0,1-16,0V136H112a8,8,0,0,1,0-16Z"/></svg>`
	iconClock     = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 256 256" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M128,24A104,104,0,1,0,232,128,104.11,104.11,0,0,0,128,24Zm0,192a88,88,0,1,1,88-88A88.1,88.1,0,0,1,128,216Zm64-88a8,8,0,0,1-8,8H128a8,8,0,0,1-8-8V72a8,8,0,0,1,16,0v48h48A8,8,0,0,1,192,128Z"/></svg>`
	iconTag       = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 256 256" fill="#666" style="vertical-align: middle; margin-right: 0.25em;"><path d="M243.31,136,144,36.69A15.86,15.86,0,0,0,132.69,32H40a8,8,0,0,0-8,8v92.69A15.86,15.86,0,0,0,36.69,144L136,243.31a16,16,0,0,0,22.63,0l84.68-84.68a16,16,0,0,0,0-22.63Zm-96,96L48,132.69V48h84.69L232,147.31ZM96,84A12,12,0,1,1,84,72,12,12,0,0,1,96,84Z"/></svg>`
	iconWarning   = `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 256 256" fill="#ea580c" style="vertical-align: middle; margin-right: 0.25em;"><path d="M236.8,188.09,149.35,36.22h0a24.76,24.76,0,0,0-42.7,0L19.2,188.09a23.51,23.51,0,0,0,0,23.72A24.35,24.35,0,0,0,40.55,224h174.9a24.35,24.35,0,0,0,21.33-12.19A23.51,23.51,0,0,0,236.8,188.09ZM120,104a8,8,0,0,1,16,0v40a8,8,0,0,1-16,0Zm8,88a12,12,0,1,1,12-12A12,12,0,0,1,128,192Z"/></svg>`
	iconSuccess   = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 256 256" fill="#059669" style="vertical-align: middle; margin-right: 0.15em;"><path d="M128,24A104,104,0,1,0,232,128,104.11,104.11,0,0,0,128,24Zm49.53,85.79-58,56a8,8,0,0,1-11.08,0l-30-28a8,8,0,0,1,10.9-11.72L114,148.71l52.42-50.5a8,8,0,0,1,11.16,11.58Z"/></svg>`
	iconFailed    = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 256 256" fill="#dc2626" style="vertical-align: middle; margin-right: 0.15em;"><path d="M128,24A104,104,0,1,0,232,128,104.11,104.11,0,0,0,128,24Zm37.66,130.34a8,8,0,0,1-11.32,11.32L128,139.31l-26.34,26.35a8,8,0,0,1-11.32-11.32L116.69,128,90.34,101.66a8,8,0,0,1,11.32-11.32L128,116.69l26.34-26.35a8,8,0,0,1,11.32,11.32L139.31,128Z"/></svg>`
	iconSkipped   = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 256 256" fill="#666" style="vertical-align: middle; margin-right: 0.15em;"><path d="M200,32V224a8,8,0,0,1-16,0V32a8,8,0,0,1,16,0ZM144.4,121.37,64.49,67.08A15.91,15.91,0,0,0,40,80.42v95.16a15.91,15.91,0,0,0,24.49,13.34l79.91-54.29a15.71,15.71,0,0,0,0-13.26Z"/></svg>`
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
