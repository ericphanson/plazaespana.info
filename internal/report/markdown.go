package report

import (
	"fmt"
	"io"
	"strings"
)

// WriteMarkdown writes a markdown-formatted report.
func (r *BuildReport) WriteMarkdown(w io.Writer) error {
	var b strings.Builder

	// Title and metadata
	b.WriteString("# Madrid Events Site Build Report\n\n")
	b.WriteString(fmt.Sprintf("**Build Time:** %s  \n", r.BuildTime.Format("2006-01-02 15:04:05 MST")))
	b.WriteString(fmt.Sprintf("**Duration:** %.2fs  \n", r.Duration.Seconds()))
	b.WriteString(fmt.Sprintf("**Exit Status:** %s  \n", statusEmoji(r.ExitStatus)+" "+r.ExitStatus))
	b.WriteString(fmt.Sprintf("**Events Generated:** %d\n\n", r.EventsCount))

	// Build pipeline diagram
	b.WriteString("## Build Pipeline\n\n")
	b.WriteString("```mermaid\n")
	b.WriteString("graph LR\n")
	b.WriteString("    A[Fetch Data] --> B[Deduplicate]\n")
	b.WriteString("    B --> C[Geo Filter]\n")
	b.WriteString("    C --> D[Time Filter]\n")
	b.WriteString("    D --> E[Render HTML]\n")
	b.WriteString("    D --> F[Render JSON]\n")
	b.WriteString(fmt.Sprintf("    A -.-> A1[%s: %d events]\n", r.Fetching.SourceUsed, r.Processing.Deduplication.Input))
	b.WriteString(fmt.Sprintf("    B -.-> B1[%d events]\n", r.Processing.Deduplication.Output))
	b.WriteString(fmt.Sprintf("    C -.-> C1[%d events]\n", r.Processing.GeoFilter.Kept))
	b.WriteString(fmt.Sprintf("    D -.-> D1[%d events]\n", r.Processing.TimeFilter.Kept))
	b.WriteString(fmt.Sprintf("    E -.-> E1[%s]\n", r.Output.HTML.Status))
	b.WriteString(fmt.Sprintf("    F -.-> F1[%s]\n", r.Output.JSON.Status))
	b.WriteString("```\n\n")

	// Data Fetching
	b.WriteString("## 1. Data Fetching\n\n")
	b.WriteString(fmt.Sprintf("**Source Used:** %s  \n", r.Fetching.SourceUsed))
	b.WriteString(fmt.Sprintf("**Total Duration:** %.2fs\n\n", r.Fetching.TotalDuration.Seconds()))

	for i, attempt := range r.Fetching.Attempts {
		b.WriteString(fmt.Sprintf("### Attempt %d: %s Source\n\n", i+1, attempt.Source))

		// Metadata table
		b.WriteString("| Property | Value |\n")
		b.WriteString("|----------|-------|\n")
		b.WriteString(fmt.Sprintf("| URL | `%s` |\n", attempt.URL))
		b.WriteString(fmt.Sprintf("| Status | %s **%s** |\n", statusEmoji(attempt.Status), attempt.Status))
		b.WriteString(fmt.Sprintf("| Duration | %.2fs |\n", attempt.Duration.Seconds()))

		if attempt.Status == "SUCCESS" {
			b.WriteString(fmt.Sprintf("| HTTP Status | %d |\n", attempt.HTTPStatus))
			if attempt.ContentType != "" {
				b.WriteString(fmt.Sprintf("| Content-Type | `%s` |\n", attempt.ContentType))
			}
			if attempt.Size > 0 {
				b.WriteString(fmt.Sprintf("| Response Size | %.2f KB |\n", float64(attempt.Size)/1024))
			}
			b.WriteString(fmt.Sprintf("| Events Parsed | %d |\n", attempt.EventCount))
		} else if attempt.Status == "FAILED" {
			b.WriteString(fmt.Sprintf("| Error Type | Parse Error |\n"))
		}

		b.WriteString("\n")

		// Error details
		if attempt.Status == "FAILED" && attempt.Error != "" {
			b.WriteString("#### Error Details\n\n")
			b.WriteString("```\n")
			b.WriteString(attempt.Error)
			b.WriteString("\n```\n\n")

			// Add specific recommendations based on error type
			if strings.Contains(attempt.Error, "invalid character") && strings.Contains(attempt.Error, "\\n") {
				b.WriteString("**Analysis:** JSON contains unescaped newline characters in string fields.\n\n")
				b.WriteString("**Recommendation:**\n")
				b.WriteString("- Implement JSON preprocessing to escape literal newlines\n")
				b.WriteString("- Or use a more lenient JSON parser\n")
				b.WriteString("- Or report data quality issue to Madrid open data portal\n\n")
			} else if strings.Contains(attempt.Error, "expected element type") && strings.Contains(attempt.Error, "Contenidos") {
				b.WriteString("**Analysis:** XML structure mismatch - root element is `<Contenidos>` not `<response>`.\n\n")
				b.WriteString("**Recommendation:**\n")
				b.WriteString("- Update `XMLResponse` struct in `internal/fetch/types.go`:\n")
				b.WriteString("  ```go\n")
				b.WriteString("  type XMLResponse struct {\n")
				b.WriteString("      XMLName xml.Name   `xml:\"Contenidos\"`\n")
				b.WriteString("      Events  []RawEvent `xml:\"contenido\"`\n")
				b.WriteString("  }\n")
				b.WriteString("  ```\n\n")
			}
		}
	}

	// Data Processing
	b.WriteString("## 2. Data Processing\n\n")

	// Deduplication
	b.WriteString("### Deduplication (by ID-EVENTO)\n\n")
	b.WriteString("| Metric | Count |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Input | %d |\n", r.Processing.Deduplication.Input))
	b.WriteString(fmt.Sprintf("| Duplicates Removed | %d |\n", r.Processing.Deduplication.Duplicates))
	b.WriteString(fmt.Sprintf("| Output | %d |\n", r.Processing.Deduplication.Output))
	b.WriteString(fmt.Sprintf("| Duration | %.3fs |\n", r.Processing.Deduplication.Duration.Seconds()))
	b.WriteString("\n")

	// Geographic filtering
	b.WriteString("### Geographic Filtering (Haversine Distance)\n\n")
	b.WriteString("**Reference Point:** Plaza de España  \n")
	b.WriteString(fmt.Sprintf("**Coordinates:** (%.5f, %.5f)  \n", r.Processing.GeoFilter.RefLat, r.Processing.GeoFilter.RefLon))
	b.WriteString(fmt.Sprintf("**Radius:** %.2f km\n\n", r.Processing.GeoFilter.Radius))

	b.WriteString("| Filter Result | Count | Percentage |\n")
	b.WriteString("|---------------|-------|------------|\n")
	b.WriteString(fmt.Sprintf("| Input | %d | 100.0%% |\n", r.Processing.GeoFilter.Input))
	b.WriteString(fmt.Sprintf("| Missing Coordinates | %d | %.1f%% |\n",
		r.Processing.GeoFilter.MissingCoords,
		percent(r.Processing.GeoFilter.MissingCoords, r.Processing.GeoFilter.Input)))
	b.WriteString(fmt.Sprintf("| Outside Radius | %d | %.1f%% |\n",
		r.Processing.GeoFilter.OutsideRadius,
		percent(r.Processing.GeoFilter.OutsideRadius, r.Processing.GeoFilter.Input)))
	b.WriteString(fmt.Sprintf("| **Kept** | **%d** | **%.1f%%** |\n",
		r.Processing.GeoFilter.Kept,
		percent(r.Processing.GeoFilter.Kept, r.Processing.GeoFilter.Input)))
	b.WriteString(fmt.Sprintf("| Duration | %.3fs | - |\n", r.Processing.GeoFilter.Duration.Seconds()))
	b.WriteString("\n")

	// Add visualization of geographic filtering
	keptPct := int(percent(r.Processing.GeoFilter.Kept, r.Processing.GeoFilter.Input))
	filteredPct := 100 - keptPct
	if keptPct > 0 {
		b.WriteString("**Distribution:**\n\n")
		b.WriteString("```mermaid\n")
		b.WriteString("pie title Geographic Filter Results\n")
		b.WriteString(fmt.Sprintf("    \"Kept (%d events)\" : %d\n", r.Processing.GeoFilter.Kept, keptPct))
		b.WriteString(fmt.Sprintf("    \"Filtered Out\" : %d\n", filteredPct))
		b.WriteString("```\n\n")
	}

	// Time filtering
	b.WriteString("### Time Filtering (Future Events)\n\n")
	b.WriteString(fmt.Sprintf("**Reference Time:** %s  \n", r.Processing.TimeFilter.ReferenceTime.Format("2006-01-02 15:04:05 MST")))
	b.WriteString(fmt.Sprintf("**Timezone:** %s\n\n", r.Processing.TimeFilter.Timezone))

	b.WriteString("| Filter Result | Count | Percentage |\n")
	b.WriteString("|---------------|-------|------------|\n")
	b.WriteString(fmt.Sprintf("| Input | %d | 100.0%% |\n", r.Processing.TimeFilter.Input))
	b.WriteString(fmt.Sprintf("| Parse Failures | %d | %.1f%% |\n",
		r.Processing.TimeFilter.ParseFailures,
		percent(r.Processing.TimeFilter.ParseFailures, r.Processing.TimeFilter.Input)))
	b.WriteString(fmt.Sprintf("| Past Events | %d | %.1f%% |\n",
		r.Processing.TimeFilter.PastEvents,
		percent(r.Processing.TimeFilter.PastEvents, r.Processing.TimeFilter.Input)))
	b.WriteString(fmt.Sprintf("| **Kept** | **%d** | **%.1f%%** |\n",
		r.Processing.TimeFilter.Kept,
		percent(r.Processing.TimeFilter.Kept, r.Processing.TimeFilter.Input)))
	b.WriteString("\n")

	// Data Quality Issues
	if len(r.DataQuality) > 0 {
		b.WriteString("## 3. Data Quality Issues\n\n")

		for _, issue := range r.DataQuality {
			symbol := "ℹ️"
			if issue.Severity == "WARNING" {
				symbol = "⚠️"
			} else if issue.Severity == "ERROR" {
				symbol = "❌"
			}

			b.WriteString(fmt.Sprintf("### %s %s\n\n", symbol, issue.Type))
			b.WriteString(fmt.Sprintf("**Severity:** %s  \n", issue.Severity))
			b.WriteString(fmt.Sprintf("**Occurrences:** %d  \n", issue.Count))
			b.WriteString(fmt.Sprintf("**Description:** %s\n\n", issue.Description))

			if len(issue.Examples) > 0 {
				b.WriteString("**Examples:**\n\n")
				for _, example := range issue.Examples {
					b.WriteString(fmt.Sprintf("- `%s`\n", example))
				}
				b.WriteString("\n")
			}

			if issue.Recommendation != "" {
				b.WriteString(fmt.Sprintf("**Recommendation:** %s\n\n", issue.Recommendation))
			}
		}
	}

	// Output Generation
	b.WriteString("## 4. Output Generation\n\n")

	writeOutputTable := func(name string, file OutputFile) {
		b.WriteString(fmt.Sprintf("### %s\n\n", name))
		b.WriteString("| Property | Value |\n")
		b.WriteString("|----------|-------|\n")
		b.WriteString(fmt.Sprintf("| Path | `%s` |\n", file.Path))
		b.WriteString(fmt.Sprintf("| Status | %s **%s** |\n", statusEmoji(file.Status), file.Status))
		if file.Status == "SUCCESS" && file.Size > 0 {
			b.WriteString(fmt.Sprintf("| File Size | %.2f KB |\n", float64(file.Size)/1024))
		}
		if file.Status == "FAILED" && file.Error != "" {
			b.WriteString(fmt.Sprintf("| Error | `%s` |\n", file.Error))
		}
		b.WriteString(fmt.Sprintf("| Duration | %.3fs |\n", file.Duration.Seconds()))
		b.WriteString("\n")
	}

	writeOutputTable("HTML Generation", r.Output.HTML)
	writeOutputTable("JSON API Generation", r.Output.JSON)

	if r.Output.Snapshot.Path != "" {
		writeOutputTable("Snapshot Saved", r.Output.Snapshot)
	}

	// Summary
	b.WriteString("## 5. Summary\n\n")

	statusSymbol := "✅"
	if r.ExitStatus == "FAILED" {
		statusSymbol = "❌"
	} else if r.ExitStatus == "PARTIAL" {
		statusSymbol = "⚠️"
	}
	b.WriteString(fmt.Sprintf("**Build Status:** %s %s\n\n", statusSymbol, r.ExitStatus))

	// Events pipeline flow
	b.WriteString("### Events Pipeline\n\n")
	b.WriteString("```\n")
	b.WriteString(fmt.Sprintf("Fetched:           %4d (from %s)\n", r.Processing.Deduplication.Input, r.Fetching.SourceUsed))
	b.WriteString(fmt.Sprintf("   ↓ dedup\n"))
	b.WriteString(fmt.Sprintf("After Dedup:       %4d (-%d duplicates)\n", r.Processing.Deduplication.Output, r.Processing.Deduplication.Duplicates))
	b.WriteString(fmt.Sprintf("   ↓ geo filter (%.2fkm)\n", r.Processing.GeoFilter.Radius))
	b.WriteString(fmt.Sprintf("After Geo Filter:  %4d (-%d outside radius, -%d missing coords)\n",
		r.Processing.GeoFilter.Kept,
		r.Processing.GeoFilter.OutsideRadius,
		r.Processing.GeoFilter.MissingCoords))
	b.WriteString(fmt.Sprintf("   ↓ time filter\n"))
	b.WriteString(fmt.Sprintf("After Time Filter: %4d (-%d past, -%d parse errors)\n",
		r.Processing.TimeFilter.Kept,
		r.Processing.TimeFilter.PastEvents,
		r.Processing.TimeFilter.ParseFailures))
	b.WriteString(fmt.Sprintf("   ↓ render\n"))
	b.WriteString(fmt.Sprintf("Final Output:      %4d events\n", r.EventsCount))
	b.WriteString("```\n\n")

	// Performance metrics
	b.WriteString("### Performance Metrics\n\n")
	fetchPct := r.Fetching.TotalDuration.Seconds() / r.Duration.Seconds() * 100
	processPct := (r.Processing.Deduplication.Duration.Seconds() +
		r.Processing.GeoFilter.Duration.Seconds()) / r.Duration.Seconds() * 100
	renderPct := (r.Output.HTML.Duration.Seconds() +
		r.Output.JSON.Duration.Seconds()) / r.Duration.Seconds() * 100

	b.WriteString("| Phase | Duration | % of Total |\n")
	b.WriteString("|-------|----------|------------|\n")
	b.WriteString(fmt.Sprintf("| Fetching | %.2fs | %.1f%% |\n", r.Fetching.TotalDuration.Seconds(), fetchPct))
	b.WriteString(fmt.Sprintf("| Processing | %.2fs | %.1f%% |\n",
		r.Processing.Deduplication.Duration.Seconds()+r.Processing.GeoFilter.Duration.Seconds(), processPct))
	b.WriteString(fmt.Sprintf("| Rendering | %.2fs | %.1f%% |\n",
		r.Output.HTML.Duration.Seconds()+r.Output.JSON.Duration.Seconds(), renderPct))
	b.WriteString(fmt.Sprintf("| **Total** | **%.2fs** | **100.0%%** |\n", r.Duration.Seconds()))
	b.WriteString("\n")

	// Warnings
	if len(r.Warnings) > 0 {
		b.WriteString("### ⚠️ Warnings\n\n")
		for i, warning := range r.Warnings {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, warning))
		}
		b.WriteString("\n")
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		b.WriteString("### 💡 Recommendations\n\n")
		for i, rec := range r.Recommendations {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("---\n\n")
	b.WriteString("*Report generated by madrid-events build system*\n")

	_, err := w.Write([]byte(b.String()))
	return err
}

// statusEmoji returns an emoji for a status string.
func statusEmoji(status string) string {
	switch status {
	case "SUCCESS":
		return "✅"
	case "FAILED":
		return "❌"
	case "PARTIAL":
		return "⚠️"
	case "SKIPPED":
		return "⏭️"
	default:
		return "❓"
	}
}
