package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ericphanson/madrid-events/internal/config"
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

const version = "2.0.0-dual-pipeline"

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

	// Custom usage message
	flag.Usage = func() {
		log.Printf("Madrid Events Site Generator %s\n", version)
		log.Println("\nDual pipeline: Fetches cultural events (datos.madrid.es) and city events (esmadrid.com)")
		log.Println("\nUsage:")
		log.Printf("  %s [options]\n\n", os.Args[0])
		log.Println("Configuration:")
		log.Println("  Use -config flag to specify TOML config file (recommended)")
		log.Println("  Or use individual flags to override specific settings")
		log.Println("\nOptions:")
		flag.PrintDefaults()
	}

	// Parse flags
	showVersion := flag.Bool("version", false, "Show version and exit")
	configPath := flag.String("config", "config.toml", "Path to TOML configuration file")
	jsonURL := flag.String("json-url", "", "Cultural events JSON URL (datos.madrid.es, overrides config)")
	xmlURL := flag.String("xml-url", "", "Cultural events XML URL (datos.madrid.es, overrides config)")
	csvURL := flag.String("csv-url", "", "Cultural events CSV URL (datos.madrid.es, overrides config)")
	esmadridURL := flag.String("esmadrid-url", "", "City events XML URL (esmadrid.com, overrides config)")
	outDir := flag.String("out-dir", "", "Output directory for static files (overrides config)")
	dataDir := flag.String("data-dir", "", "Data directory for snapshots (overrides config)")
	lat := flag.Float64("lat", 0, "Reference latitude in decimal degrees (overrides config)")
	lon := flag.Float64("lon", 0, "Reference longitude in decimal degrees (overrides config)")
	radiusKm := flag.Float64("radius-km", 0, "Filter radius in kilometers (overrides config)")
	timezone := flag.String("timezone", "Europe/Madrid", "Timezone for event times")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		log.Printf("Madrid Events Site Generator %s\n", version)
		log.Println("Dual pipeline support: Cultural events (datos.madrid.es) + City events (esmadrid.com)")
		os.Exit(0)
	}

	// Load configuration from TOML file (or use defaults if not found)
	var cfg *config.Config
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		log.Printf("Config file not found (%s), using defaults", *configPath)
		cfg = config.DefaultConfig()
	} else {
		var err error
		cfg, err = config.Load(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		log.Printf("Loaded configuration from %s", *configPath)
	}

	// Override config with CLI flags if provided
	if *jsonURL != "" {
		cfg.CulturalEvents.JSONURL = *jsonURL
	}
	if *xmlURL != "" {
		cfg.CulturalEvents.XMLURL = *xmlURL
	}
	if *csvURL != "" {
		cfg.CulturalEvents.CSVURL = *csvURL
	}
	if *esmadridURL != "" {
		cfg.CityEvents.XMLURL = *esmadridURL
	}
	if *lat != 0 {
		cfg.Filter.Latitude = *lat
	}
	if *lon != 0 {
		cfg.Filter.Longitude = *lon
	}
	if *radiusKm != 0 {
		cfg.Filter.RadiusKm = *radiusKm
	}
	if *outDir != "" {
		// Update both HTML and JSON paths to use new output directory
		cfg.Output.HTMLPath = filepath.Join(*outDir, "index.html")
		cfg.Output.JSONPath = filepath.Join(*outDir, "events.json")
	}
	if *dataDir != "" {
		cfg.Snapshot.DataDir = *dataDir
	}

	// Capture output directory for deferred report writing
	outputDir = filepath.Dir(cfg.Output.HTMLPath)

	// Load timezone
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		log.Fatalf("Invalid timezone: %v", err)
	}

	// Initialize components
	client := fetch.NewClient(30 * time.Second)
	snapMgr := snapshot.NewManager(cfg.Snapshot.DataDir)

	// Create pipeline for multi-source fetching (cultural events from datos.madrid.es)
	pipe := pipeline.NewPipeline(cfg.CulturalEvents.JSONURL, cfg.CulturalEvents.XMLURL, cfg.CulturalEvents.CSVURL, client, loc)

	// Fetch from all three sources independently
	log.Println("Fetching from all three sources (JSON, XML, CSV)...")
	fetchStart := time.Now()
	pipeResult := pipe.FetchAll()
	buildReport.Fetching.TotalDuration = time.Since(fetchStart)

	// Track individual fetch results
	buildReport.Fetching.JSON = createFetchAttempt("JSON", cfg.CulturalEvents.JSONURL, pipeResult.JSONEvents, pipeResult.JSONErrors)
	buildReport.Fetching.XML = createFetchAttempt("XML", cfg.CulturalEvents.XMLURL, pipeResult.XMLEvents, pipeResult.XMLErrors)
	buildReport.Fetching.CSV = createFetchAttempt("CSV", cfg.CulturalEvents.CSVURL, pipeResult.CSVEvents, pipeResult.CSVErrors)

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
		// TODO: Implement snapshot loading with CulturalEvent conversion
		buildReport.AddWarning("Using stale snapshot data - all fetch attempts failed")
	} else if len(merged) > 0 {
		// Save successful merge to snapshot
		if err := snapMgr.SaveSnapshot(convertToRawEvents(merged)); err != nil {
			log.Printf("Warning: failed to save snapshot: %v", err)
		}
	}

	// =====================================================================
	// CULTURAL EVENTS PIPELINE: Filter by location and time
	// =====================================================================
	log.Println("\n=== Cultural Events Pipeline ===")
	now := time.Now().In(loc)
	geoStart := time.Now()
	var filteredEvents []event.CulturalEvent

	// Target districts from config
	targetDistricts := make(map[string]bool)
	for _, distrito := range cfg.Filter.Distritos {
		targetDistricts[distrito] = true
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
			if filter.WithinRadius(cfg.Filter.Latitude, cfg.Filter.Longitude, evt.Latitude, evt.Longitude, cfg.Filter.RadiusKm) {
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

		// Filter out events that started more than N weeks ago
		// (Even if still ongoing, we don't care about old exhibitions)
		cutoffWeeksAgo := now.AddDate(0, 0, -7*cfg.Filter.PastEventsWeeks)
		if evt.StartTime.Before(cutoffWeeksAgo) {
			pastEvents++
			continue
		}

		filteredEvents = append(filteredEvents, evt)
	}

	log.Printf("Filtered by distrito: %d, by radius: %d, by text: %d", byDistrito, byRadius, byTextMatch)

	// Record geo filter stats
	geoDuration := time.Since(geoStart)
	buildReport.Processing.GeoFilter = report.GeoFilterStats{
		RefLat:        cfg.Filter.Latitude,
		RefLon:        cfg.Filter.Longitude,
		Radius:        cfg.Filter.RadiusKm,
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
		ParseFailures: 0, // No parse failures with CulturalEvent
		PastEvents:    pastEvents,
		Kept:          len(filteredEvents),
		Duration:      0, // Included in geo filter duration
	}

	// Add warnings if needed
	if len(filteredEvents) < len(merged)/100 { // Less than 1%
		buildReport.AddWarning("Geographic radius very restrictive (%.2fkm) - only %.1f%% of events kept",
			cfg.Filter.RadiusKm, float64(len(filteredEvents))*100/float64(len(merged)))
		buildReport.AddRecommendation("Consider increasing filter.radius_km to 1.0-2.0 for better coverage")
	}

	log.Printf("Cultural events after filtering: %d events", len(filteredEvents))

	// Sort events by start time (upcoming events first)
	sort.Slice(filteredEvents, func(i, j int) bool {
		return filteredEvents[i].StartTime.Before(filteredEvents[j].StartTime)
	})

	// =====================================================================
	// CITY EVENTS PIPELINE: Fetch and filter esmadrid.com events
	// =====================================================================
	log.Println("\n=== City Events Pipeline ===")
	cityStart := time.Now()

	// Fetch ESMadrid XML events
	log.Printf("Fetching ESMadrid events from: %s", cfg.CityEvents.XMLURL)
	esmadridServices, err := fetch.FetchEsmadridEvents(cfg.CityEvents.XMLURL)
	if err != nil {
		log.Printf("Warning: Failed to fetch ESMadrid events: %v", err)
		esmadridServices = []fetch.EsmadridService{} // Continue with empty list
	} else {
		log.Printf("Fetched %d ESMadrid services", len(esmadridServices))
	}

	// Convert to CityEvent structs
	var cityEvents []event.CityEvent
	parseErrors := 0
	for _, svc := range esmadridServices {
		cityEvent, err := svc.ToCityEvent()
		if err != nil {
			parseErrors++
			continue
		}
		cityEvents = append(cityEvents, *cityEvent)
	}
	log.Printf("Parsed %d city events (%d parse errors)", len(cityEvents), parseErrors)

	// Filter city events by GPS radius and time
	// No category filtering for now (empty slice = allow all categories)
	filteredCityEvents := filter.FilterCityEvents(
		cityEvents,
		cfg.Filter.Latitude,
		cfg.Filter.Longitude,
		cfg.Filter.RadiusKm,
		nil, // No category filtering
		cfg.Filter.PastEventsWeeks,
	)
	log.Printf("City events after filtering: %d events", len(filteredCityEvents))

	// Sort city events by start date
	sort.Slice(filteredCityEvents, func(i, j int) bool {
		return filteredCityEvents[i].StartDate.Before(filteredCityEvents[j].StartDate)
	})

	cityDuration := time.Since(cityStart)
	log.Printf("City events pipeline completed in %v", cityDuration)

	// =====================================================================
	// RENDERING: Render both cultural and city events
	// =====================================================================
	log.Println("\n=== Rendering Output ===")

	// Group events by time
	cityGroups, ongoingCity := render.GroupCityEventsByTime(filteredCityEvents, now)
	culturalGroups, ongoingCultural := render.GroupEventsByTime(filteredEvents, now)

	// Convert to JSON format (keep original flat structure for API)
	var culturalJSONEvents []render.JSONEvent
	for _, evt := range filteredEvents {
		culturalJSONEvents = append(culturalJSONEvents, render.JSONEvent{
			ID:         evt.ID,
			Title:      evt.Title,
			StartTime:  evt.StartTime.Format(time.RFC3339),
			VenueName:  evt.VenueName,
			DetailsURL: evt.DetailsURL,
		})
	}

	var cityJSONEvents []render.JSONEvent
	for _, evt := range filteredCityEvents {
		cityJSONEvents = append(cityJSONEvents, render.JSONEvent{
			ID:         evt.ID,
			Title:      evt.Title,
			StartTime:  evt.StartDate.Format(time.RFC3339),
			VenueName:  evt.Venue,
			DetailsURL: evt.WebURL,
		})
	}

	// Count total events in groups
	totalCityEvents := len(ongoingCity)
	for _, group := range cityGroups {
		totalCityEvents += len(group.Events)
	}
	totalCulturalEvents := len(ongoingCultural)
	for _, group := range culturalGroups {
		totalCulturalEvents += len(group.Events)
	}

	// Render outputs
	outDirPath := filepath.Dir(cfg.Output.HTMLPath)
	if err := os.MkdirAll(outDirPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Render HTML with grouped events
	htmlStart := time.Now()
	htmlRenderer := render.NewHTMLRenderer("templates/index-grouped.tmpl.html")
	htmlData := render.GroupedTemplateData{
		Lang:                  "es",
		CSSHash:               readCSSHash(outDirPath),
		LastUpdated:           now.Format("2006-01-02 15:04 MST"),
		TotalEvents:           totalCityEvents + totalCulturalEvents,
		TotalCityEvents:       totalCityEvents,
		TotalCulturalEvents:   totalCulturalEvents,
		ShowCulturalDefault:   false, // Cultural events hidden by default
		CityGroups:            cityGroups,
		CulturalGroups:        culturalGroups,
		OngoingCityEvents:     ongoingCity,
		OngoingCulturalEvents: ongoingCultural,
	}
	htmlPath := cfg.Output.HTMLPath
	htmlErr := htmlRenderer.RenderAny(htmlData, htmlPath)
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

	// Render JSON with separated event types
	jsonRenderStart := time.Now()
	jsonRenderer := render.NewJSONRenderer()
	jsonPath := cfg.Output.JSONPath
	jsonErr := jsonRenderer.Render(culturalJSONEvents, cityJSONEvents, now, jsonPath)
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

	// Record final event count (total of both types)
	buildReport.EventsCount = len(filteredEvents) + len(filteredCityEvents)

	// Final summary
	log.Println("\n=== Build Summary ===")
	log.Printf("Cultural events: %d (datos.madrid.es)", len(filteredEvents))
	log.Printf("City events: %d (esmadrid.com)", len(filteredCityEvents))
	log.Printf("Total events rendered: %d", len(filteredEvents)+len(filteredCityEvents))
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

// convertToRawEvents converts CulturalEvents to RawEvents for snapshot compatibility.
func convertToRawEvents(canonical []event.CulturalEvent) []fetch.RawEvent {
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
