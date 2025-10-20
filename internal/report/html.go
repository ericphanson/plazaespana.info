package report

import (
	"fmt"
	"io"
	"strings"
)

// WriteHTML writes an HTML-formatted build report.
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
      }
    }

    * { box-sizing: border-box; }

    body {
      margin: 0;
      background: var(--bg);
      color: var(--fg);
      font: 16px/1.55 system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, "Helvetica Neue", Arial;
    }

    .container {
      max-width: 1000px;
      margin: 0 auto;
      padding: 2rem;
    }

    h1, h2, h3 { margin: 1.5rem 0 0.75rem; }
    h1 { font-size: 2rem; }
    h2 { font-size: 1.5rem; margin-top: 2rem; }

    .meta {
      color: var(--muted);
      font-size: 0.9rem;
      margin: 0.5rem 0;
    }

    .stat-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 1rem;
      margin: 1.5rem 0;
    }

    .stat-card {
      background: var(--card);
      padding: 1.25rem;
      border-radius: 8px;
    }

    .stat-label {
      color: var(--muted);
      font-size: 0.85rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      margin-bottom: 0.5rem;
    }

    .stat-value {
      font-size: 2rem;
      font-weight: 600;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      margin: 1rem 0;
      background: var(--card);
      border-radius: 8px;
      overflow: hidden;
    }

    th, td {
      text-align: left;
      padding: 0.75rem 1rem;
      border-bottom: 1px solid var(--border);
    }

    th {
      background: var(--card);
      font-weight: 600;
      color: var(--muted);
      font-size: 0.85rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    tr:last-child td {
      border-bottom: none;
    }

    .status-success { color: var(--success); }
    .status-failure { color: var(--failure); }

    .warning {
      background: var(--card);
      border-left: 4px solid var(--failure);
      padding: 1rem;
      margin: 1rem 0;
      border-radius: 4px;
    }

    .recommendation {
      background: var(--card);
      border-left: 4px solid var(--link);
      padding: 1rem;
      margin: 1rem 0;
      border-radius: 4px;
    }

    .back-link {
      display: inline-block;
      margin-top: 2rem;
      color: var(--link);
      text-decoration: none;
    }

    .back-link:hover {
      text-decoration: underline;
    }
  </style>
</head>
<body>
<div class="container">
`)

	// Title and metadata
	b.WriteString("<h1>Build Report</h1>\n")
	b.WriteString(fmt.Sprintf("<p class=\"meta\"><strong>Build Time:</strong> %s</p>\n",
		r.BuildTime.Format("2006-01-02 15:04:05 MST")))
	b.WriteString(fmt.Sprintf("<p class=\"meta\"><strong>Duration:</strong> %.2fs</p>\n",
		r.Duration.Seconds()))
	b.WriteString(fmt.Sprintf("<p class=\"meta\"><strong>Status:</strong> %s</p>\n",
		r.ExitStatus))

	// Key statistics
	b.WriteString("<div class=\"stat-grid\">\n")
	b.WriteString("  <div class=\"stat-card\">\n")
	b.WriteString("    <div class=\"stat-label\">Events Generated</div>\n")
	b.WriteString(fmt.Sprintf("    <div class=\"stat-value\">%d</div>\n", r.EventsCount))
	b.WriteString("  </div>\n")

	if r.Processing.Merge.TotalBeforeMerge > 0 {
		b.WriteString("  <div class=\"stat-card\">\n")
		b.WriteString("    <div class=\"stat-label\">Total Fetched</div>\n")
		b.WriteString(fmt.Sprintf("    <div class=\"stat-value\">%d</div>\n", r.Processing.Merge.TotalBeforeMerge))
		b.WriteString("  </div>\n")

		b.WriteString("  <div class=\"stat-card\">\n")
		b.WriteString("    <div class=\"stat-label\">Unique Events</div>\n")
		b.WriteString(fmt.Sprintf("    <div class=\"stat-value\">%d</div>\n", r.Processing.Merge.UniqueEvents))
		b.WriteString("  </div>\n")

		b.WriteString("  <div class=\"stat-card\">\n")
		b.WriteString("    <div class=\"stat-label\">Duplicates Removed</div>\n")
		b.WriteString(fmt.Sprintf("    <div class=\"stat-value\">%d</div>\n", r.Processing.Merge.Duplicates))
		b.WriteString("  </div>\n")
	}
	b.WriteString("</div>\n")

	// Data Sources
	b.WriteString("<h2>Data Sources</h2>\n")
	b.WriteString("<table>\n")
	b.WriteString("  <thead>\n")
	b.WriteString("    <tr><th>Source</th><th>Status</th><th>Events</th><th>Duration</th></tr>\n")
	b.WriteString("  </thead>\n")
	b.WriteString("  <tbody>\n")

	for _, attempt := range []FetchAttempt{r.Fetching.JSON, r.Fetching.XML, r.Fetching.CSV} {
		if attempt.Source == "" {
			continue
		}
		statusClass := "status-success"
		statusText := "✅ " + attempt.Status
		if attempt.Status != "SUCCESS" {
			statusClass = "status-failure"
			statusText = "❌ " + attempt.Status
		}
		b.WriteString(fmt.Sprintf("    <tr><td>%s</td><td class=\"%s\">%s</td><td>%d</td><td>%.2fs</td></tr>\n",
			attempt.Source, statusClass, statusText, attempt.EventCount, attempt.Duration.Seconds()))
	}

	b.WriteString("  </tbody>\n")
	b.WriteString("</table>\n")

	// Merge & Deduplication
	if r.Processing.Merge.TotalBeforeMerge > 0 {
		b.WriteString("<h2>Merge & Deduplication</h2>\n")
		dedupPercent := 0.0
		if r.Processing.Merge.TotalBeforeMerge > 0 {
			dedupPercent = float64(r.Processing.Merge.Duplicates) * 100.0 / float64(r.Processing.Merge.TotalBeforeMerge)
		}
		b.WriteString(fmt.Sprintf("<p>Merged %d events from 3 sources into %d unique events (%.1f%% deduplication rate)</p>\n",
			r.Processing.Merge.TotalBeforeMerge, r.Processing.Merge.UniqueEvents, dedupPercent))

		b.WriteString("<table>\n")
		b.WriteString("  <thead>\n")
		b.WriteString("    <tr><th>Coverage</th><th>Count</th></tr>\n")
		b.WriteString("  </thead>\n")
		b.WriteString("  <tbody>\n")
		b.WriteString(fmt.Sprintf("    <tr><td>In all 3 sources</td><td>%d</td></tr>\n", r.Processing.Merge.InAllThree))
		b.WriteString(fmt.Sprintf("    <tr><td>In 2 sources</td><td>%d</td></tr>\n", r.Processing.Merge.InTwoSources))
		b.WriteString(fmt.Sprintf("    <tr><td>In 1 source only</td><td>%d</td></tr>\n", r.Processing.Merge.InOneSource))
		b.WriteString("  </tbody>\n")
		b.WriteString("</table>\n")
	}

	// Geographic Filtering
	if r.Processing.GeoFilter.Input > 0 {
		b.WriteString("<h2>Geographic Filtering</h2>\n")
		b.WriteString(fmt.Sprintf("<p>Reference: %.5f°N, %.5f°W • Radius: %.2f km</p>\n",
			r.Processing.GeoFilter.RefLat, -r.Processing.GeoFilter.RefLon, r.Processing.GeoFilter.Radius))

		b.WriteString("<table>\n")
		b.WriteString("  <thead>\n")
		b.WriteString("    <tr><th>Category</th><th>Count</th></tr>\n")
		b.WriteString("  </thead>\n")
		b.WriteString("  <tbody>\n")
		b.WriteString(fmt.Sprintf("    <tr><td>Input events</td><td>%d</td></tr>\n", r.Processing.GeoFilter.Input))
		b.WriteString(fmt.Sprintf("    <tr><td>Missing coordinates</td><td>%d</td></tr>\n", r.Processing.GeoFilter.MissingCoords))
		b.WriteString(fmt.Sprintf("    <tr><td>Outside radius</td><td>%d</td></tr>\n", r.Processing.GeoFilter.OutsideRadius))
		b.WriteString(fmt.Sprintf("    <tr><td>Kept</td><td>%d</td></tr>\n", r.Processing.GeoFilter.Kept))
		b.WriteString("  </tbody>\n")
		b.WriteString("</table>\n")
	}

	// Time Filtering
	if r.Processing.TimeFilter.Input > 0 {
		b.WriteString("<h2>Time Filtering</h2>\n")
		b.WriteString(fmt.Sprintf("<p>Reference time: %s (%s)</p>\n",
			r.Processing.TimeFilter.ReferenceTime.Format("2006-01-02 15:04:05"),
			r.Processing.TimeFilter.Timezone))

		b.WriteString("<table>\n")
		b.WriteString("  <thead>\n")
		b.WriteString("    <tr><th>Category</th><th>Count</th></tr>\n")
		b.WriteString("  </thead>\n")
		b.WriteString("  <tbody>\n")
		b.WriteString(fmt.Sprintf("    <tr><td>Input events</td><td>%d</td></tr>\n", r.Processing.TimeFilter.Input))
		b.WriteString(fmt.Sprintf("    <tr><td>Past events</td><td>%d</td></tr>\n", r.Processing.TimeFilter.PastEvents))
		b.WriteString(fmt.Sprintf("    <tr><td>Kept (future)</td><td>%d</td></tr>\n", r.Processing.TimeFilter.Kept))
		b.WriteString("  </tbody>\n")
		b.WriteString("</table>\n")
	}

	// Warnings
	if len(r.Warnings) > 0 {
		b.WriteString("<h2>Warnings</h2>\n")
		for _, warning := range r.Warnings {
			b.WriteString(fmt.Sprintf("<div class=\"warning\">⚠️ %s</div>\n", warning))
		}
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		b.WriteString("<h2>Recommendations</h2>\n")
		for _, rec := range r.Recommendations {
			b.WriteString(fmt.Sprintf("<div class=\"recommendation\">💡 %s</div>\n", rec))
		}
	}

	// Output files
	b.WriteString("<h2>Output Files</h2>\n")
	b.WriteString("<table>\n")
	b.WriteString("  <thead>\n")
	b.WriteString("    <tr><th>File</th><th>Status</th><th>Size</th><th>Duration</th></tr>\n")
	b.WriteString("  </thead>\n")
	b.WriteString("  <tbody>\n")

	if r.Output.HTML.Path != "" {
		statusClass := "status-success"
		statusText := "✅ " + r.Output.HTML.Status
		if r.Output.HTML.Status != "SUCCESS" {
			statusClass = "status-failure"
			statusText = "❌ " + r.Output.HTML.Status
		}
		size := fmt.Sprintf("%d bytes", r.Output.HTML.Size)
		if r.Output.HTML.Size > 1024 {
			size = fmt.Sprintf("%.1f KB", float64(r.Output.HTML.Size)/1024.0)
		}
		b.WriteString(fmt.Sprintf("    <tr><td>%s</td><td class=\"%s\">%s</td><td>%s</td><td>%.3fs</td></tr>\n",
			r.Output.HTML.Path, statusClass, statusText, size, r.Output.HTML.Duration.Seconds()))
	}

	if r.Output.JSON.Path != "" {
		statusClass := "status-success"
		statusText := "✅ " + r.Output.JSON.Status
		if r.Output.JSON.Status != "SUCCESS" {
			statusClass = "status-failure"
			statusText = "❌ " + r.Output.JSON.Status
		}
		size := fmt.Sprintf("%d bytes", r.Output.JSON.Size)
		if r.Output.JSON.Size > 1024 {
			size = fmt.Sprintf("%.1f KB", float64(r.Output.JSON.Size)/1024.0)
		}
		b.WriteString(fmt.Sprintf("    <tr><td>%s</td><td class=\"%s\">%s</td><td>%s</td><td>%.3fs</td></tr>\n",
			r.Output.JSON.Path, statusClass, statusText, size, r.Output.JSON.Duration.Seconds()))
	}

	b.WriteString("  </tbody>\n")
	b.WriteString("</table>\n")

	// Back link
	b.WriteString("<a href=\"index.html\" class=\"back-link\">← Back to Events</a>\n")

	// Close HTML
	b.WriteString("</div>\n</body>\n</html>\n")

	_, err := w.Write([]byte(b.String()))
	return err
}
