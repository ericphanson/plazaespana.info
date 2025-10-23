package report

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Emoji icon constants for build report
const (
	iconTheater   = "🎭"
	iconCelebrate = "🎉"
	iconBroadcast = "📍"
	iconSync      = "🔄"
	iconMap       = "🗺️"
	iconTarget    = "🎯"
	iconClock     = "⏰"
	iconTag       = "🏷️"
	iconWarning   = "⚠️"
	iconSuccess   = "✅"
	iconFailed    = "❌"
	iconSkipped   = "⏭️"
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
`, iconCelebrate, r.CityPipeline.Name, r.CityPipeline.Source, r.CityPipeline.EventCount, formatDuration(r.CityPipeline.Duration)))

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
`, iconCelebrate, iconBroadcast))

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

	// Weather Integration
	if r.Weather != nil {
		b.WriteString(`    <h2>⛅ Weather Integration</h2>
    <div class="section">
`)
		statusSymbol := iconSuccess
		if r.Weather.Error != "" {
			statusSymbol = iconWarning
		} else if !r.Weather.APIKeyPresent {
			statusSymbol = iconWarning
		}

		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>%sStatus</span>
        <span>%s</span>
      </div>
`, statusSymbol, func() string {
			if r.Weather.Error != "" {
				return "Failed: " + r.Weather.Error
			}
			if !r.Weather.APIKeyPresent {
				return "API key not set"
			}
			return "Success"
		}()))

		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Municipality</span>
        <span>%s</span>
      </div>
`, r.Weather.Municipality))

		if r.Weather.DaysCovered > 0 {
			b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Forecast Days</span>
        <span>%d days</span>
      </div>
`, r.Weather.DaysCovered))
		}

		if r.Weather.EventsMatched > 0 || r.Weather.EventsUnmatched > 0 {
			total := r.Weather.EventsMatched + r.Weather.EventsUnmatched
			pct := 0
			if total > 0 {
				pct = (r.Weather.EventsMatched * 100) / total
			}
			b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Events with Weather</span>
        <span>%d / %d (%d%%)</span>
      </div>
`, r.Weather.EventsMatched, total, pct))
		}

		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>Duration</span>
        <span>%s</span>
      </div>
`, formatDuration(r.Weather.Duration)))

		b.WriteString(`    </div>
`)
	}

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
