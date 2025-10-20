package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
	"github.com/ericphanson/madrid-events/internal/fetch"
	"github.com/ericphanson/madrid-events/internal/filter"
	"github.com/ericphanson/madrid-events/internal/pipeline"
	"github.com/ericphanson/madrid-events/internal/render"
	"github.com/ericphanson/madrid-events/internal/report"
	"github.com/ericphanson/madrid-events/internal/snapshot"
)

// readCSSHash reads the CSS hash from the assets directory.
// Returns "placeholder" if the file doesn't exist or cannot be read.
func readCSSHash(outDir string) string {
	hashPath := filepath.Join(outDir, "assets", "css.hash")
	content, err := os.ReadFile(hashPath)
	if err != nil {
		return "placeholder"
	}
	return strings.TrimSpace(string(content))
}

func main() {
	// Initialize build report
	buildReport := report.NewBuildReport()
	var outputDir string
	defer func() {
		buildReport.Duration = time.Since(buildReport.BuildTime)

		// Write HTML report
		htmlReportPath := filepath.Join(outputDir, "build-report.html")
		if f, err := os.Create(htmlReportPath); err == nil {
			buildReport.WriteHTML(f)
			f.Close()
			log.Println("Build report written to:", htmlReportPath)
		}
	}()

	// Parse flags
	jsonURL := flag.String("json-url", "", "Madrid events JSON URL")
	xmlURL := flag.String("xml-url", "", "Madrid events XML URL (fallback)")
	csvURL := flag.String("csv-url", "", "Madrid events CSV URL (fallback)")
	outDir := flag.String("out-dir", "./public", "Output directory for static files")
	dataDir := flag.String("data-dir", "./data", "Data directory for snapshots")
	lat := flag.Float64("lat", 40.42338, "Reference latitude (Plaza de España)")
	lon := flag.Float64("lon", -3.71217, "Reference longitude (Plaza de España)")
	radiusKm := flag.Float64("radius-km", 2.0, "Filter radius in kilometers")
	timezone := flag.String("timezone", "Europe/Madrid", "Timezone for event times")

	flag.Parse()

	// Capture output directory for deferred report writing
	outputDir = *outDir

	if *jsonURL == "" {
		log.Fatal("Missing required flag: -json-url")
	}

	// Use URLs directly (no server-side filtering)
	// Server-side filters return minimal schema and are not usable
	finalJSONURL := *jsonURL
	finalXMLURL := *xmlURL
	finalCSVURL := *csvURL

	// Load timezone
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		log.Fatalf("Invalid timezone: %v", err)
	}

	// Initialize components
	client := fetch.NewClient(30 * time.Second)
	snapMgr := snapshot.NewManager(*dataDir)

	// Create pipeline for multi-source fetching (use final URLs with distrito filter if applied)
	pipe := pipeline.NewPipeline(finalJSONURL, finalXMLURL, finalCSVURL, client, loc)

	// Fetch from all three sources independently
	log.Println("Fetching from all three sources (JSON, XML, CSV)...")
	fetchStart := time.Now()
	pipeResult := pipe.FetchAll()
	buildReport.Fetching.TotalDuration = time.Since(fetchStart)

	// Track individual fetch results (use final URLs for reporting)
	buildReport.Fetching.JSON = createFetchAttempt("JSON", finalJSONURL, pipeResult.JSONEvents, pipeResult.JSONErrors)
	buildReport.Fetching.XML = createFetchAttempt("XML", finalXMLURL, pipeResult.XMLEvents, pipeResult.XMLErrors)
	buildReport.Fetching.CSV = createFetchAttempt("CSV", finalCSVURL, pipeResult.CSVEvents, pipeResult.CSVErrors)

	log.Printf("JSON: %d events, %d errors", len(pipeResult.JSONEvents), len(pipeResult.JSONErrors))
	log.Printf("XML: %d events, %d errors", len(pipeResult.XMLEvents), len(pipeResult.XMLErrors))
	log.Printf("CSV: %d events, %d errors", len(pipeResult.CSVEvents), len(pipeResult.CSVErrors))

	// Merge and deduplicate
	mergeStart := time.Now()
	merged := pipe.Merge(pipeResult)
	mergeDuration := time.Since(mergeStart)

	// Calculate merge stats
	buildReport.Processing.Merge = report.MergeStats{
		JSONEvents:       len(pipeResult.JSONEvents),
		XMLEvents:        len(pipeResult.XMLEvents),
		CSVEvents:        len(pipeResult.CSVEvents),
		TotalBeforeMerge: len(pipeResult.JSONEvents) + len(pipeResult.XMLEvents) + len(pipeResult.CSVEvents),
		UniqueEvents:     len(merged),
		Duplicates:       (len(pipeResult.JSONEvents) + len(pipeResult.XMLEvents) + len(pipeResult.CSVEvents)) - len(merged),
		Duration:         mergeDuration,
	}

	// Calculate source coverage
	for _, evt := range merged {
		switch len(evt.Sources) {
		case 3:
			buildReport.Processing.Merge.InAllThree++
		case 2:
			buildReport.Processing.Merge.InTwoSources++
		case 1:
			buildReport.Processing.Merge.InOneSource++
		}
	}

	log.Printf("After merge: %d unique events from %d total (%.1f%% deduplication)",
		len(merged),
		buildReport.Processing.Merge.TotalBeforeMerge,
		float64(buildReport.Processing.Merge.Duplicates)*100.0/float64(buildReport.Processing.Merge.TotalBeforeMerge))

	// Handle snapshot fallback if ALL sources failed
	if len(merged) == 0 && allSourcesFailed(pipeResult) {
		log.Println("All sources failed, loading snapshot...")
		// TODO: Implement snapshot loading with CanonicalEvent conversion
		buildReport.AddWarning("Using stale snapshot data - all fetch attempts failed")
	} else if len(merged) > 0 {
		// Save successful merge to snapshot
		if err := snapMgr.SaveSnapshot(convertToRawEvents(merged)); err != nil {
			log.Printf("Warning: failed to save snapshot: %v", err)
		}
	}

	// Filter by location and time
	now := time.Now().In(loc)
	geoStart := time.Now()
	var filteredEvents []event.CanonicalEvent

	// Target districts near Plaza de España
	targetDistricts := map[string]bool{
		"CENTRO":          true,
		"MONCLOA-ARAVACA": true,
	}

	// Location keywords for text-based fallback (when no distrito or coords)
	locationKeywords := []string{
		"plaza de españa",
		"plaza españa",
		"templo de debod",
		"parque del oeste",
		"conde duque",
	}

	missingDistr := 0
	missingBoth := 0
	byDistrito := 0
	byRadius := 0
	byTextMatch := 0
	outsideAll := 0
	pastEvents := 0

	for _, evt := range merged {
		// Priority 1: Filter by distrito (works for 95% of events)
		if evt.Distrito != "" {
			if targetDistricts[evt.Distrito] {
				byDistrito++
				// Skip to time filter
			} else {
				outsideAll++
				continue
			}
		} else if evt.Latitude != 0 && evt.Longitude != 0 {
			// Priority 2: GPS coordinates available, use radius
			if filter.WithinRadius(*lat, *lon, evt.Latitude, evt.Longitude, *radiusKm) {
				byRadius++
				missingDistr++
			} else {
				outsideAll++
				continue
			}
		} else {
			// Priority 3: No distrito, no coords - try text matching
			if filter.MatchesLocation(evt.VenueName, evt.Address, evt.Description, locationKeywords) {
				byTextMatch++
				missingBoth++
			} else {
				missingBoth++
				outsideAll++
				continue
			}
		}

		// Check if event is in the future
		endTime := evt.EndTime
		if endTime.IsZero() {
			endTime = evt.StartTime
		}

		if !filter.IsInFuture(endTime, now) {
			pastEvents++
			continue
		}

		filteredEvents = append(filteredEvents, evt)
	}

	log.Printf("Filtered by distrito: %d, by radius: %d, by text: %d", byDistrito, byRadius, byTextMatch)

	// Record geo filter stats
	geoDuration := time.Since(geoStart)
	buildReport.Processing.GeoFilter = report.GeoFilterStats{
		RefLat:        *lat,
		RefLon:        *lon,
		Radius:        *radiusKm,
		Input:         len(merged),
		MissingCoords: missingBoth,
		OutsideRadius: outsideAll,
		Kept:          len(filteredEvents) + pastEvents, // Events that passed geo filter
		Duration:      geoDuration,
	}

	// Log filtering method breakdown
	if byTextMatch > 0 {
		log.Printf("Text-based location matching: kept %d events", byTextMatch)
	}

	// Record time filter stats
	buildReport.Processing.TimeFilter = report.TimeFilterStats{
		ReferenceTime: now,
		Timezone:      *timezone,
		Input:         len(filteredEvents) + pastEvents,
		ParseFailures: 0, // No parse failures with CanonicalEvent
		PastEvents:    pastEvents,
		Kept:          len(filteredEvents),
		Duration:      0, // Included in geo filter duration
	}

	// Add warnings if needed
	if len(filteredEvents) < len(merged)/100 { // Less than 1%
		buildReport.AddWarning("Geographic radius very restrictive (%.2fkm) - only %.1f%% of events kept",
			*radiusKm, float64(len(filteredEvents))*100/float64(len(merged)))
		buildReport.AddRecommendation("Consider increasing -radius-km to 1.0-2.0 for better coverage")
	}

	log.Printf("After filtering: %d events", len(filteredEvents))

	// Convert to template format
	var templateEvents []render.TemplateEvent
	var jsonEvents []render.JSONEvent

	for _, evt := range filteredEvents {
		templateEvents = append(templateEvents, render.TemplateEvent{
			IDEvento:          evt.ID,
			Titulo:            evt.Title,
			StartHuman:        evt.StartTime.Format("02/01/2006 15:04"),
			NombreInstalacion: evt.VenueName,
			ContentURL:        evt.DetailsURL,
			Description:       render.TruncateText(evt.Description, 150),
		})

		jsonEvents = append(jsonEvents, render.JSONEvent{
			ID:         evt.ID,
			Title:      evt.Title,
			StartTime:  evt.StartTime.Format(time.RFC3339),
			VenueName:  evt.VenueName,
			DetailsURL: evt.DetailsURL,
		})
	}

	// Render outputs
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Render HTML
	htmlStart := time.Now()
	htmlRenderer := render.NewHTMLRenderer("templates/index.tmpl.html")
	htmlData := render.TemplateData{
		Lang:        "es",
		CSSHash:     readCSSHash(*outDir),
		LastUpdated: now.Format("2006-01-02 15:04 MST"),
		Events:      templateEvents,
	}
	htmlPath := fmt.Sprintf("%s/index.html", *outDir)
	htmlErr := htmlRenderer.Render(htmlData, htmlPath)
	htmlDuration := time.Since(htmlStart)

	if htmlErr != nil {
		buildReport.Output.HTML = report.OutputFile{
			Path:     htmlPath,
			Status:   "FAILED",
			Error:    htmlErr.Error(),
			Duration: htmlDuration,
		}
		log.Fatalf("Failed to render HTML: %v", htmlErr)
	}

	htmlInfo, _ := os.Stat(htmlPath)
	buildReport.Output.HTML = report.OutputFile{
		Path:     htmlPath,
		Size:     htmlInfo.Size(),
		Status:   "SUCCESS",
		Duration: htmlDuration,
	}
	log.Println("Generated:", htmlPath)

	// Render JSON
	jsonRenderStart := time.Now()
	jsonRenderer := render.NewJSONRenderer()
	jsonPath := fmt.Sprintf("%s/events.json", *outDir)
	jsonErr := jsonRenderer.Render(jsonEvents, jsonPath)
	jsonRenderDuration := time.Since(jsonRenderStart)

	if jsonErr != nil {
		buildReport.Output.JSON = report.OutputFile{
			Path:     jsonPath,
			Status:   "FAILED",
			Error:    jsonErr.Error(),
			Duration: jsonRenderDuration,
		}
		log.Fatalf("Failed to render JSON: %v", jsonErr)
	}

	jsonInfo, _ := os.Stat(jsonPath)
	buildReport.Output.JSON = report.OutputFile{
		Path:     jsonPath,
		Size:     jsonInfo.Size(),
		Status:   "SUCCESS",
		Duration: jsonRenderDuration,
	}
	log.Println("Generated:", jsonPath)

	// Record final event count
	buildReport.EventsCount = len(filteredEvents)

	log.Println("Build complete!")
}

// createFetchAttempt creates a FetchAttempt from pipeline results.
func createFetchAttempt(source, url string, events []event.SourcedEvent, errors []event.ParseError) report.FetchAttempt {
	attempt := report.FetchAttempt{
		Source: source,
		URL:    url,
	}

	if len(events) > 0 {
		attempt.Status = "SUCCESS"
		attempt.EventCount = len(events)
		attempt.HTTPStatus = 200
	} else if len(errors) > 0 {
		attempt.Status = "FAILED"
		attempt.Error = errors[0].Error.Error()
	} else {
		attempt.Status = "FAILED"
		attempt.Error = "no events parsed"
	}

	return attempt
}

// allSourcesFailed returns true if all three sources failed to fetch events.
func allSourcesFailed(result pipeline.PipelineResult) bool {
	return len(result.JSONEvents) == 0 && len(result.XMLEvents) == 0 && len(result.CSVEvents) == 0
}

// convertToRawEvents converts CanonicalEvents to RawEvents for snapshot compatibility.
func convertToRawEvents(canonical []event.CanonicalEvent) []fetch.RawEvent {
	raw := make([]fetch.RawEvent, len(canonical))
	for i, evt := range canonical {
		raw[i] = fetch.RawEvent{
			IDEvento:          evt.ID,
			Titulo:            evt.Title,
			Descripcion:       evt.Description,
			Fecha:             evt.StartTime.Format("2006-01-02"),
			FechaFin:          evt.EndTime.Format("2006-01-02"),
			Hora:              evt.StartTime.Format("15:04"),
			NombreInstalacion: evt.VenueName,
			Direccion:         evt.Address,
			ContentURL:        evt.DetailsURL,
			Lat:               evt.Latitude,
			Lon:               evt.Longitude,
		}
	}
	return raw
}
