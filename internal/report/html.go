package report

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// WriteHTML writes an HTML-formatted build report for dual pipeline architecture.
func (r *BuildReport) WriteHTML(w io.Writer) error {
	var b strings.Builder

	// HTML header with embedded CSS
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Build Report - Madrid Events</title>
  <style>
    :root {
      --bg: #ffffff;
      --fg: #111;
      --muted: #666;
      --card: #f6f6f6;
      --link: #0645ad;
      --border: #ddd;
      --success: #0a8754;
      --failure: #d93025;
      --cultural: #7c3aed;
      --city: #ea580c;
    }

    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #0f1115;
        --fg: #eaeaea;
        --muted: #9aa0a6;
        --card: #1a1d24;
        --link: #8ab4f8;
        --border: #444;
        --success: #34a853;
        --failure: #f28b82;
        --cultural: #a78bfa;
        --city: #fb923c;
      }
    }

    * { box-sizing: border-box; }

    body {
      margin: 0;
      background: var(--bg);
      color: var(--fg);
      font: 16px/1.6 system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif;
    }

    header, main, footer {
      max-width: 1200px;
      margin: auto;
      padding: 1rem;
    }

    h1 { margin: 0; font-size: 2rem; }
    h2 { margin: 1.5rem 0 0.5rem; font-size: 1.5rem; border-bottom: 2px solid var(--border); padding-bottom: 0.5rem; }
    h3 { margin: 1rem 0 0.5rem; font-size: 1.2rem; }

    .summary-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1.5rem;
      margin: 1rem 0;
    }

    .summary-card h2 { margin-top: 0; border-bottom: none; }

    .summary-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 1rem;
      margin-top: 1rem;
    }

    .summary-item strong { display: block; color: var(--muted); font-size: 0.875rem; margin-bottom: 0.25rem; }
    .summary-item span { font-size: 1.5rem; font-weight: 600; }

    .pipeline-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
      gap: 1.5rem;
      margin: 1rem 0;
    }

    .pipeline-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1.5rem;
    }

    .pipeline-card.cultural { border-left: 4px solid var(--cultural); }
    .pipeline-card.city { border-left: 4px solid var(--city); }

    .pipeline-header {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 1rem;
    }

    .pipeline-header h3 { margin: 0; }
    .pipeline-header .icon { font-size: 1.5rem; }

    .pipeline-stat {
      display: flex;
      justify-content: space-between;
      padding: 0.5rem 0;
      border-bottom: 1px solid var(--border);
    }

    .pipeline-stat:last-child { border-bottom: none; }

    .section {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1.5rem;
      margin: 1rem 0;
    }

    .section h3 { margin-top: 0; }

    .metric-row {
      display: flex;
      justify-content: space-between;
      padding: 0.5rem 0;
    }

    .metric-row span:first-child { color: var(--muted); }

    .status-success { color: var(--success); font-weight: 600; }
    .status-failure { color: var(--failure); font-weight: 600; }

    .fetch-attempts {
      margin: 0.5rem 0;
      padding-left: 1.5rem;
    }

    .fetch-attempts li {
      margin: 0.25rem 0;
    }

    .warning-box {
      background: rgba(234, 88, 12, 0.1);
      border-left: 4px solid var(--city);
      padding: 1rem;
      margin: 1rem 0;
      border-radius: 4px;
    }

    .warning-box ul {
      margin: 0.5rem 0;
      padding-left: 1.5rem;
    }

    footer {
      color: var(--muted);
      font-size: 0.875rem;
      text-align: center;
      margin-top: 2rem;
    }
  </style>
</head>
<body>
  <header>
    <h1>Build Report</h1>
    <p style="color: var(--muted); margin: 0.5rem 0;">Madrid Events Site Generator</p>
  </header>

  <main>
`)

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
          <span class="icon">🎭</span>
          <h3 style="color: var(--cultural);">%s</h3>
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
`, r.CulturalPipeline.Name, r.CulturalPipeline.Source, r.CulturalPipeline.EventCount, formatDuration(r.CulturalPipeline.Duration)))

	// City Pipeline Card
	b.WriteString(fmt.Sprintf(`      <div class="pipeline-card city">
        <div class="pipeline-header">
          <span class="icon">🎉</span>
          <h3 style="color: var(--city);">%s</h3>
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
`, r.CityPipeline.Name, r.CityPipeline.Source, r.CityPipeline.EventCount, formatDuration(r.CityPipeline.Duration)))

	b.WriteString(`    </div>
`)

	// Cultural Events Pipeline Detailed
	b.WriteString(fmt.Sprintf(`    <h2 style="color: var(--cultural);">🎭 Cultural Events Pipeline</h2>
    <div class="section">
      <h3>📡 Data Fetching</h3>
`))

	for _, attempt := range r.CulturalPipeline.Fetching.Attempts {
		statusSymbol := "✅"
		if attempt.Status == "FAILED" {
			statusSymbol = "❌"
		} else if attempt.Status == "SKIPPED" {
			statusSymbol = "⏭️"
		}
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>%s %s</span>
        <span>%s</span>
      </div>
`, statusSymbol, attempt.Source, formatAttempt(attempt)))
	}

	if r.CulturalPipeline.Merging != nil {
		b.WriteString(`      <h3>🔄 Deduplication</h3>
`)
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
		b.WriteString(`      <h3>🗺️ Distrito Filtering</h3>
`)
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
		b.WriteString(`      <h3>🎯 Geographic Filtering</h3>
`)
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
		b.WriteString(`      <h3>⏰ Time Filtering</h3>
`)
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
	b.WriteString(fmt.Sprintf(`    <h2 style="color: var(--city);">🎉 City Events Pipeline</h2>
    <div class="section">
      <h3>📡 Data Fetching</h3>
`))

	for _, attempt := range r.CityPipeline.Fetching.Attempts {
		statusSymbol := "✅"
		if attempt.Status == "FAILED" {
			statusSymbol = "❌"
		} else if attempt.Status == "SKIPPED" {
			statusSymbol = "⏭️"
		}
		b.WriteString(fmt.Sprintf(`      <div class="metric-row">
        <span>%s %s</span>
        <span>%s</span>
      </div>
`, statusSymbol, attempt.Source, formatAttempt(attempt)))
	}

	// City Filtering
	if r.CityPipeline.Filtering.GeoFilter != nil {
		b.WriteString(`      <h3>🎯 Geographic Filtering</h3>
`)
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
		b.WriteString(`      <h3>🏷️ Category Filtering</h3>
`)
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
		b.WriteString(`      <h3>⏰ Time Filtering</h3>
`)
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
		b.WriteString(`    <div class="warning-box">
      <h3 style="margin-top: 0;">⚠️ Warnings</h3>
      <ul>
`)
		for _, warning := range r.Warnings {
			b.WriteString(fmt.Sprintf("        <li>%s</li>\n", warning))
		}
		b.WriteString(`      </ul>
    </div>
`)
	}

	// Footer
	b.WriteString(`  </main>

  <footer>
    <p>Generated by Madrid Events Site Generator</p>
    <p><a href="/">← Back to events</a></p>
  </footer>
</body>
</html>`)

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
