package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ericphanson/plazaespana.info/internal/audit"
	"github.com/ericphanson/plazaespana.info/internal/config"
	"github.com/ericphanson/plazaespana.info/internal/event"
	"github.com/ericphanson/plazaespana.info/internal/fetch"
	"github.com/ericphanson/plazaespana.info/internal/filter"
	"github.com/ericphanson/plazaespana.info/internal/pipeline"
	"github.com/ericphanson/plazaespana.info/internal/render"
	"github.com/ericphanson/plazaespana.info/internal/report"
	"github.com/ericphanson/plazaespana.info/internal/snapshot"
	"github.com/ericphanson/plazaespana.info/internal/version"
	"github.com/ericphanson/plazaespana.info/internal/weather"
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

// readBuildReportCSSHash reads the build report CSS hash from the assets directory.
// Returns "placeholder" if the file doesn't exist or cannot be read.
func readBuildReportCSSHash(outDir string) string {
	hashPath := filepath.Join(outDir, "assets", "build-report-css.hash")
	content, err := os.ReadFile(hashPath)
	if err != nil {
		return "placeholder"
	}
	return strings.TrimSpace(string(content))
}

const buildVersion = "2.0.0-dual-pipeline"

func main() {
	// Initialize build report
	buildReport := report.NewBuildReport()
	var outputDir string
	var reportBasePath string
	defer func() {
		buildReport.Duration = time.Since(buildReport.BuildTime)

		// Write HTML report
		htmlReportPath := filepath.Join(outputDir, "build-report.html")
		if f, err := os.Create(htmlReportPath); err == nil {
			reportCSSHash := readBuildReportCSSHash(outputDir)
			buildReport.WriteHTML(f, reportCSSHash, reportBasePath)
			f.Close()
			log.Println("Build report written to:", htmlReportPath)
		}
	}()

	// Custom usage message
	flag.Usage = func() {
		log.Printf("Madrid Events Site Generator %s\n", buildVersion)
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
	aemetBaseURL := flag.String("aemet-base-url", "", "AEMET API base URL (for testing, overrides default)")
	outDir := flag.String("out-dir", "", "Output directory for static files (overrides config)")
	dataDir := flag.String("data-dir", "", "Data directory for snapshots (overrides config)")
	lat := flag.Float64("lat", 0, "Reference latitude in decimal degrees (overrides config)")
	lon := flag.Float64("lon", 0, "Reference longitude in decimal degrees (overrides config)")
	radiusKm := flag.Float64("radius-km", 0, "Filter radius in kilometers (overrides config)")
	timezone := flag.String("timezone", "Europe/Madrid", "Timezone for event times")
	fetchMode := flag.String("fetch-mode", "development", "Fetch mode: production or development (affects caching/throttling)")
	templatePath := flag.String("template-path", "generator/templates/index.tmpl.html", "Path to HTML template file")
	basePath := flag.String("base-path", "", "Base path for URLs (e.g., /previews/PR5 for preview deployments)")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		log.Printf("Madrid Events Site Generator %s\n", buildVersion)
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

	// Capture output directory and base path for deferred report writing
	outputDir = filepath.Dir(cfg.Output.HTMLPath)
	reportBasePath = *basePath

	// Load timezone
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		log.Fatalf("Invalid timezone: %v", err)
	}

	// Initialize components
	// Parse fetch mode and get config
	mode := fetch.ParseMode(*fetchMode)
	var modeConfig fetch.ModeConfig
	if mode == fetch.ProductionMode {
		modeConfig = fetch.DefaultProductionConfig()
	} else {
		modeConfig = fetch.DefaultDevelopmentConfig()
	}

	// Create HTTP cache directory
	cacheDir := filepath.Join(cfg.Snapshot.DataDir, "http-cache")

	// Create client with respectful fetching support
	client, err := fetch.NewClient(30*time.Second, modeConfig, cacheDir)
	if err != nil {
		log.Fatalf("Failed to create fetch client: %v", err)
	}
	log.Printf("Fetch mode: %s (cache TTL: %v, min delay: %v)", mode, modeConfig.CacheTTL, modeConfig.MinDelay)

	snapMgr := snapshot.NewManager(cfg.Snapshot.DataDir)

	// Initialize cultural pipeline report
	buildReport.CulturalPipeline.Name = "Cultural Events"
	buildReport.CulturalPipeline.Source = "datos.madrid.es"
	culturalStart := time.Now()

	// Create pipeline for multi-source fetching (cultural events from datos.madrid.es)
	pipe := pipeline.NewPipeline(cfg.CulturalEvents.JSONURL, cfg.CulturalEvents.XMLURL, cfg.CulturalEvents.CSVURL, client, loc)

	// Fetch from all three sources independently
	log.Println("Fetching from all three sources (JSON, XML, CSV)...")
	fetchStart := time.Now()
	pipeResult := pipe.FetchAll()
	buildReport.CulturalPipeline.Fetching.TotalDuration = time.Since(fetchStart)

	// Track individual fetch attempts
	buildReport.CulturalPipeline.Fetching.Attempts = []report.FetchAttempt{
		createFetchAttempt("JSON", cfg.CulturalEvents.JSONURL, pipeResult.JSONEvents, pipeResult.JSONErrors),
		createFetchAttempt("XML", cfg.CulturalEvents.XMLURL, pipeResult.XMLEvents, pipeResult.XMLErrors),
		createFetchAttempt("CSV", cfg.CulturalEvents.CSVURL, pipeResult.CSVEvents, pipeResult.CSVErrors),
	}

	log.Printf("JSON: %d events, %d errors", len(pipeResult.JSONEvents), len(pipeResult.JSONErrors))
	log.Printf("XML: %d events, %d errors", len(pipeResult.XMLEvents), len(pipeResult.XMLErrors))
	log.Printf("CSV: %d events, %d errors", len(pipeResult.CSVEvents), len(pipeResult.CSVErrors))

	// Merge and deduplicate
	mergeStart := time.Now()
	merged := pipe.Merge(pipeResult)
	mergeDuration := time.Since(mergeStart)

	// Calculate merge stats for cultural pipeline
	mergeStats := report.MergeStats{
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
			mergeStats.InAllThree++
		case 2:
			mergeStats.InTwoSources++
		case 1:
			mergeStats.InOneSource++
		}
	}

	buildReport.CulturalPipeline.Merging = &mergeStats

	log.Printf("After merge: %d unique events from %d total (%.1f%% deduplication)",
		len(merged),
		mergeStats.TotalBeforeMerge,
		float64(mergeStats.Duplicates)*100.0/float64(mergeStats.TotalBeforeMerge))

	// Handle snapshot fallback if ALL sources failed
	if len(merged) == 0 && allSourcesFailed(pipeResult) {
		log.Println("All sources failed, attempting to load snapshot...")

		// Load snapshot
		snapshot, err := snapMgr.LoadSnapshot()
		if err != nil {
			log.Printf("Warning: Failed to load snapshot: %v", err)
			buildReport.AddWarning("All fetch sources failed and no snapshot available")
		} else {
			log.Printf("Loaded snapshot with %d events", len(snapshot))

			// Convert RawEvent back to CulturalEvent
			snapshotEvents := make([]event.CulturalEvent, 0, len(snapshot))
			for _, raw := range snapshot {
				// Parse times
				startTime, err := time.ParseInLocation("2006-01-02 15:04", raw.Fecha+" "+raw.Hora, loc)
				if err != nil {
					// Try without time if parsing fails
					startTime, err = time.ParseInLocation("2006-01-02", raw.Fecha, loc)
					if err != nil {
						log.Printf("Warning: Failed to parse snapshot event %s time: %v", raw.IDEvento, err)
						continue
					}
				}

				endTime, err := time.ParseInLocation("2006-01-02", raw.FechaFin, loc)
				if err != nil {
					// Use start time if end time parsing fails
					endTime = startTime
				}

				// Convert to CulturalEvent
				canonical := event.CulturalEvent{
					ID:          raw.IDEvento,
					Title:       raw.Titulo,
					Description: raw.Descripcion,
					StartTime:   startTime,
					EndTime:     endTime,
					VenueName:   raw.NombreInstalacion,
					Address:     raw.Direccion,
					DetailsURL:  raw.ContentURL,
					Latitude:    raw.Lat,
					Longitude:   raw.Lon,
					Sources:     []string{"SNAPSHOT"}, // Mark as from snapshot
				}

				snapshotEvents = append(snapshotEvents, canonical)
			}

			log.Printf("Converted %d snapshot events to CulturalEvent", len(snapshotEvents))
			merged = snapshotEvents
			buildReport.AddWarning("Using snapshot data - all fetch attempts failed (snapshot has %d events)", len(snapshotEvents))
		}
	} else if len(merged) > 0 {
		// Save successful merge to snapshot
		if err := snapMgr.SaveSnapshot(convertToRawEvents(merged)); err != nil {
			log.Printf("Warning: failed to save snapshot: %v", err)
		}
	}

	// =====================================================================
	// CULTURAL EVENTS PIPELINE: Tag events with filter decisions
	// =====================================================================
	log.Println("\n=== Cultural Events Pipeline ===")
	now := time.Now().In(loc)
	geoStart := time.Now()

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

	// Step 1: Evaluate all filters for all events and record results
	// Non-destructive: Keep ALL events in memory
	allEvents := make([]event.CulturalEvent, 0, len(merged))
	for _, evt := range merged {
		result := event.FilterResult{}

		// Evaluate distrito filter
		result.HasDistrito = (evt.Distrito != "")
		result.Distrito = evt.Distrito
		if result.HasDistrito {
			result.DistritoMatched = targetDistricts[evt.Distrito]
		}

		// Evaluate GPS filter
		result.HasCoordinates = (evt.Latitude != 0 && evt.Longitude != 0)
		if result.HasCoordinates {
			result.GPSDistanceKm = filter.HaversineDistance(
				cfg.Filter.Latitude, cfg.Filter.Longitude,
				evt.Latitude, evt.Longitude)
			result.WithinRadius = (result.GPSDistanceKm <= cfg.Filter.RadiusKm)
		}

		// Evaluate text matching
		result.TextMatched = filter.MatchesLocation(
			evt.VenueName, evt.Address, evt.Description, locationKeywords)

		// Evaluate time filter
		result.StartDate = evt.StartTime
		result.EndDate = evt.EndTime
		result.DaysOld = int(now.Sub(evt.StartTime).Hours() / 24)
		cutoffWeeksAgo := now.AddDate(0, 0, -7*cfg.Filter.PastEventsWeeks)
		result.TooOld = evt.StartTime.Before(cutoffWeeksAgo)

		// Decide if kept (priority order: distrito -> GPS -> time -> kept)
		if result.HasDistrito && !result.DistritoMatched {
			result.Kept = false
			result.FilterReason = "outside target distrito"
		} else if result.HasCoordinates && !result.WithinRadius && !result.HasDistrito {
			result.Kept = false
			result.FilterReason = "outside GPS radius"
		} else if result.TooOld {
			result.Kept = false
			result.FilterReason = "event too old"
		} else {
			result.Kept = true
			result.FilterReason = "kept"
		}

		evt.FilterResult = result
		allEvents = append(allEvents, evt) // Keep ALL events
	}

	// Count events by filter reason (no double-counting, no mixing)
	var (
		keptEvents         = 0
		outsideDistrito    = 0
		outsideRadius      = 0
		missingCoords      = 0
		tooOld             = 0
		byDistrito         = 0
		byRadius           = 0
		byTextMatch        = 0
		missingBothSources = 0
	)

	for _, evt := range allEvents {
		switch evt.FilterResult.FilterReason {
		case "kept":
			keptEvents++
			// Count by location method (for logging)
			if evt.FilterResult.HasDistrito && evt.FilterResult.DistritoMatched {
				byDistrito++
			} else if evt.FilterResult.HasCoordinates && evt.FilterResult.WithinRadius {
				byRadius++
			} else {
				// No distrito, no coords - included by default
				missingBothSources++
				if evt.FilterResult.TextMatched {
					byTextMatch++
				}
			}
		case "outside target distrito":
			outsideDistrito++
		case "outside GPS radius":
			outsideRadius++
		case "missing location data":
			missingCoords++
		case "event too old":
			tooOld++
		}
	}

	// Step 2: Separate kept events for rendering
	var filteredEvents []event.CulturalEvent
	for _, evt := range allEvents {
		if evt.FilterResult.Kept {
			filteredEvents = append(filteredEvents, evt)
		}
	}

	log.Printf("Filtered by distrito: %d, by radius: %d, by text: %d", byDistrito, byRadius, byTextMatch)

	// Record filtering stats for cultural pipeline
	geoDuration := time.Since(geoStart)

	// Distrito filter stats (most events have distrito)
	if len(cfg.Filter.Distritos) > 0 {
		buildReport.CulturalPipeline.Filtering.DistrictoFilter = &report.DistrictoFilterStats{
			AllowedDistricts: cfg.Filter.Distritos,
			Input:            len(allEvents),
			Filtered:         outsideDistrito, // FIXED: Only "outside target distrito" events
			Kept:             keptEvents,      // FIXED: Only kept events
			Duration:         geoDuration,
		}
	}

	// Geo filter stats (for events without distrito)
	buildReport.CulturalPipeline.Filtering.GeoFilter = &report.GeoFilterStats{
		RefLat:        cfg.Filter.Latitude,
		RefLon:        cfg.Filter.Longitude,
		Radius:        cfg.Filter.RadiusKm,
		Input:         len(allEvents),
		MissingCoords: missingCoords, // FIXED: Only events with "missing location data" reason
		OutsideRadius: outsideRadius, // FIXED: Only "outside GPS radius" events
		Kept:          keptEvents,    // FIXED: Only kept events
		Duration:      geoDuration,
	}

	// Log filtering method breakdown
	if byTextMatch > 0 {
		log.Printf("Text-based location matching: kept %d events", byTextMatch)
	}

	// Time filter stats for cultural pipeline
	buildReport.CulturalPipeline.Filtering.TimeFilter = &report.TimeFilterStats{
		ReferenceTime: now,
		Timezone:      *timezone,
		Input:         len(allEvents),
		ParseFailures: 0,          // No parse failures with CulturalEvent
		PastEvents:    tooOld,     // FIXED: Only "event too old" events
		Kept:          keptEvents, // FIXED: Only kept events
		Duration:      0,          // Included in geo filter duration
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

	// Set cultural pipeline totals
	buildReport.CulturalPipeline.EventCount = len(filteredEvents)
	buildReport.CulturalPipeline.Duration = time.Since(culturalStart)

	// =====================================================================
	// CITY EVENTS PIPELINE: Fetch and filter esmadrid.com events
	// =====================================================================
	log.Println("\n=== City Events Pipeline ===")

	// Initialize city pipeline report
	buildReport.CityPipeline.Name = "City Events"
	buildReport.CityPipeline.Source = "esmadrid.com"
	cityStart := time.Now()

	// Fetch ESMadrid XML events
	log.Printf("Fetching ESMadrid events from: %s", cfg.CityEvents.XMLURL)
	cityFetchStart := time.Now()
	esmadridServices, err := fetch.FetchEsmadridEvents(cfg.CityEvents.XMLURL)
	cityFetchDuration := time.Since(cityFetchStart)

	// Track city events fetch attempt
	cityFetchAttempt := report.FetchAttempt{
		Source:   "XML",
		URL:      cfg.CityEvents.XMLURL,
		Duration: cityFetchDuration,
	}

	if err != nil {
		log.Printf("Warning: Failed to fetch ESMadrid events: %v", err)
		esmadridServices = []fetch.EsmadridService{} // Continue with empty list
		cityFetchAttempt.Status = "FAILED"
		cityFetchAttempt.Error = err.Error()
	} else {
		log.Printf("Fetched %d ESMadrid services", len(esmadridServices))
		cityFetchAttempt.Status = "SUCCESS"
		cityFetchAttempt.HTTPStatus = 200
		cityFetchAttempt.EventCount = len(esmadridServices)
	}

	buildReport.CityPipeline.Fetching.Attempts = []report.FetchAttempt{cityFetchAttempt}
	buildReport.CityPipeline.Fetching.TotalDuration = cityFetchDuration

	// Convert to CityEvent structs
	var cityEvents []event.CityEvent
	var cityParseErrors []event.ParseError
	for i, svc := range esmadridServices {
		cityEvent, err := svc.ToCityEvent()
		if err != nil {
			// Track parse error with details
			cityParseErrors = append(cityParseErrors, event.ParseError{
				Source:      "ESMadrid",
				Index:       i,
				RawData:     "", // ESMadrid services are complex, skip raw data
				Error:       err,
				RecoverType: "skipped",
			})
			continue
		}
		cityEvents = append(cityEvents, *cityEvent)
	}
	log.Printf("Parsed %d city events (%d parse errors)", len(cityEvents), len(cityParseErrors))

	// Update fetch attempt with parsed event count (not fetched services count)
	if err == nil { // Only update if fetch succeeded
		cityFetchAttempt.EventCount = len(cityEvents)

		// Add note if there were parse errors
		if len(cityParseErrors) > 0 {
			cityFetchAttempt.Error = fmt.Sprintf("Parsed %d/%d services successfully",
				len(cityEvents), len(esmadridServices))
		}

		// Update the report with corrected fetch attempt
		buildReport.CityPipeline.Fetching.Attempts = []report.FetchAttempt{cityFetchAttempt}
	}

	// Track filtering start
	cityFilterStart := time.Now()

	// Step 1: Evaluate all filters for all city events and record results
	// Non-destructive: Keep ALL events in memory
	cutoffTime := now.Add(-time.Duration(cfg.Filter.PastEventsWeeks) * 7 * 24 * time.Hour)
	allCityEvents := make([]event.CityEvent, 0, len(cityEvents))

	// Stats counters for city events
	cityOutsideRadius := 0
	cityTooOld := 0
	cityMissingCoords := 0
	cityMultiVenueKept := 0

	for _, evt := range cityEvents {
		result := event.FilterResult{}

		// Check if coordinates are actually present (not zero)
		hasCoords := evt.Latitude != 0.0 && evt.Longitude != 0.0
		result.HasCoordinates = hasCoords

		// City events don't have distrito
		result.HasDistrito = false

		// Time filter
		result.StartDate = evt.StartDate
		result.EndDate = evt.EndDate
		result.DaysOld = int(now.Sub(evt.EndDate).Hours() / 24)
		result.TooOld = evt.EndDate.Before(cutoffTime)

		// Check for Plaza de España text mention (city events only)
		result.PlazaEspanaText = filter.MatchesPlazaEspana(
			evt.Title,
			evt.Venue,
			evt.Address,
			evt.Description,
		)

		// Decide if kept (priority: missing coords -> geo/text -> too old -> kept)
		if !hasCoords {
			// No coordinates - check text matching
			if result.PlazaEspanaText {
				if result.TooOld {
					result.Kept = false
					result.FilterReason = "event too old"
					cityTooOld++
				} else {
					result.Kept = true
					result.FilterReason = "kept (multi-venue: Plaza de España)"
					result.MultiVenueKept = true
					cityMultiVenueKept++
				}
			} else {
				result.Kept = false
				result.FilterReason = "missing location data"
				cityMissingCoords++
			}
		} else {
			// Have coordinates - check geo first, then text
			result.GPSDistanceKm = filter.HaversineDistance(
				cfg.Filter.Latitude, cfg.Filter.Longitude,
				evt.Latitude, evt.Longitude)
			result.WithinRadius = (result.GPSDistanceKm <= cfg.Filter.RadiusKm)

			if result.WithinRadius {
				// Kept by geo (preferred)
				if result.TooOld {
					result.Kept = false
					result.FilterReason = "event too old"
					cityTooOld++
				} else {
					result.Kept = true
					result.FilterReason = "kept"
				}
			} else if result.PlazaEspanaText {
				// Outside radius but mentions Plaza de España
				if result.TooOld {
					result.Kept = false
					result.FilterReason = "event too old"
					cityTooOld++
				} else {
					result.Kept = true
					result.FilterReason = "kept (multi-venue: Plaza de España)"
					result.MultiVenueKept = true
					cityMultiVenueKept++
				}
			} else {
				// Outside radius and no text match
				result.Kept = false
				result.FilterReason = "outside GPS radius"
				cityOutsideRadius++
			}
		}

		evt.FilterResult = result
		allCityEvents = append(allCityEvents, evt) // Keep ALL events
	}

	// Step 2: Separate kept events for rendering
	var filteredCityEvents []event.CityEvent
	for _, evt := range allCityEvents {
		if evt.FilterResult.Kept {
			filteredCityEvents = append(filteredCityEvents, evt)
		}
	}

	cityFilterDuration := time.Since(cityFilterStart)

	log.Printf("City events after filtering: %d events (%d by geo, %d by Plaza de España text match)",
		len(filteredCityEvents), len(filteredCityEvents)-cityMultiVenueKept, cityMultiVenueKept)

	// Geo filter stats for city pipeline (FIXED: use correct counters)
	buildReport.CityPipeline.Filtering.GeoFilter = &report.GeoFilterStats{
		RefLat:         cfg.Filter.Latitude,
		RefLon:         cfg.Filter.Longitude,
		Radius:         cfg.Filter.RadiusKm,
		Input:          len(allCityEvents),
		MissingCoords:  cityMissingCoords, // FIXED: Track actual missing coordinates
		OutsideRadius:  cityOutsideRadius, // FIXED: Only "outside GPS radius" events
		Kept:           len(filteredCityEvents),
		MultiVenueKept: cityMultiVenueKept, // NEW: Count of events kept via Plaza de España text match
		Duration:       cityFilterDuration,
	}

	// Time filter stats for city pipeline (included in geo filter duration)
	buildReport.CityPipeline.Filtering.TimeFilter = &report.TimeFilterStats{
		ReferenceTime: now,
		Timezone:      *timezone,
		Input:         len(allCityEvents),
		ParseFailures: 0,
		PastEvents:    cityTooOld, // FIXED: Only "event too old" events
		Kept:          len(filteredCityEvents),
		Duration:      0, // Included in geo filter duration
	}

	// Category filter stats (currently disabled, but track for completeness)
	// buildReport.CityPipeline.Filtering.CategoryFilter would go here if enabled

	// Sort city events by start date
	sort.Slice(filteredCityEvents, func(i, j int) bool {
		return filteredCityEvents[i].StartDate.Before(filteredCityEvents[j].StartDate)
	})

	// Set city pipeline totals
	buildReport.CityPipeline.EventCount = len(filteredCityEvents)
	buildReport.CityPipeline.Duration = time.Since(cityStart)
	log.Printf("City events pipeline completed in %v", buildReport.CityPipeline.Duration)

	// =====================================================================
	// AUDIT EXPORT: Save complete audit trail with all events
	// =====================================================================
	log.Println("\n=== Exporting Audit Trail ===")

	// Collect all cultural parse errors
	culturalParseErrors := []event.ParseError{}
	culturalParseErrors = append(culturalParseErrors, pipeResult.JSONErrors...)
	culturalParseErrors = append(culturalParseErrors, pipeResult.XMLErrors...)
	culturalParseErrors = append(culturalParseErrors, pipeResult.CSVErrors...)

	auditPath := filepath.Join(cfg.Snapshot.DataDir, "audit-events.json")
	auditErr := audit.SaveAuditJSON(
		allEvents,
		allCityEvents,
		culturalParseErrors,
		cityParseErrors,
		auditPath,
		buildReport.BuildTime,
		buildReport.Duration,
	)
	if auditErr != nil {
		log.Printf("Warning: Failed to save audit JSON: %v", auditErr)
		buildReport.AddWarning("Failed to export audit trail: %v", auditErr)
	} else {
		auditInfo, _ := os.Stat(auditPath)
		totalParseErrors := len(culturalParseErrors) + len(cityParseErrors)
		log.Printf("Audit trail exported: %s (%.1f MB, %d events, %d parse errors)",
			auditPath,
			float64(auditInfo.Size())/1024/1024,
			len(allEvents)+len(allCityEvents),
			totalParseErrors)
	}

	// =====================================================================
	// WEATHER: Fetch weather forecast
	// =====================================================================
	var weatherMap map[string]*render.Weather
	log.Println("\n=== Fetching Weather ===")
	weatherStart := time.Now()

	// Initialize weather report
	buildReport.Weather = &report.WeatherReport{
		FetchTimestamp: time.Now(),
		Municipality:   cfg.Weather.MunicipalityCode,
	}

	// Get API key - try file first, then environment variable
	var apiKey string

	// Try reading from file first (preferred for production)
	if cfg.Weather.APIKeyFile != "" {
		keyBytes, err := os.ReadFile(cfg.Weather.APIKeyFile)
		if err != nil {
			log.Printf("Warning: Could not read API key file %s: %v", cfg.Weather.APIKeyFile, err)
		} else {
			apiKey = strings.TrimSpace(string(keyBytes))
			log.Printf("Loaded AEMET API key from file: %s", cfg.Weather.APIKeyFile)
		}
	}

	// Fall back to environment variable if file not available
	if apiKey == "" && cfg.Weather.APIKeyEnv != "" {
		apiKey = os.Getenv(cfg.Weather.APIKeyEnv)
		if apiKey != "" {
			log.Printf("Loaded AEMET API key from environment: %s", cfg.Weather.APIKeyEnv)
		}
	}

	buildReport.Weather.APIKeyPresent = (apiKey != "")

	if apiKey == "" {
		var keySourceMsg string
		if cfg.Weather.APIKeyFile != "" {
			keySourceMsg = fmt.Sprintf("file %s or env %s", cfg.Weather.APIKeyFile, cfg.Weather.APIKeyEnv)
		} else {
			keySourceMsg = fmt.Sprintf("env %s", cfg.Weather.APIKeyEnv)
		}
		fmt.Fprintf(os.Stderr, "Warning: AEMET API key not found (%s) - continuing without weather forecasts\n", keySourceMsg)
		log.Printf("Warning: AEMET API key not found (%s) - continuing without weather forecasts", keySourceMsg)
		buildReport.Weather.Error = fmt.Sprintf("API key not found (%s)", keySourceMsg)
	} else {
		// Determine AEMET base URL: flag overrides config
		baseURL := cfg.Weather.APIBaseURL
		if *aemetBaseURL != "" {
			baseURL = *aemetBaseURL
			log.Printf("Using AEMET base URL from flag: %s", baseURL)
		} else {
			log.Printf("Using AEMET base URL from config: %s", baseURL)
		}

		// Create weather client
		weatherClient := weather.NewClientWithBaseURL(apiKey, cfg.Weather.MunicipalityCode, client, baseURL)

		// Wait 2 seconds before weather fetch (respectful delay)
		log.Println("Waiting 2 seconds before weather fetch (respectful delay)...")
		time.Sleep(2 * time.Second)

		// Fetch forecast
		log.Printf("Fetching 7-day forecast for municipality %s...", cfg.Weather.MunicipalityCode)
		forecast, err := weatherClient.FetchForecast()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Weather fetch failed: %v (continuing without weather)\n", err)
			log.Printf("Warning: Weather fetch failed: %v (continuing without weather)", err)
			buildReport.AddWarning("Weather fetch failed: %v", err)
			buildReport.Weather.Error = err.Error()
		} else {
			log.Printf("Weather forecast received: %d days", len(forecast.Prediction.Days))
			buildReport.Weather.DaysCovered = len(forecast.Prediction.Days)

			// Build weather map for fast lookup by date
			weatherMap = weather.BuildWeatherMap(forecast, *basePath)
			log.Printf("Weather map built: %d dates", len(weatherMap))
		}
	}
	buildReport.Weather.Duration = time.Since(weatherStart)

	// =====================================================================
	// RENDERING: Render both cultural and city events
	// =====================================================================
	log.Println("\n=== Rendering Output ===")

	// Group events by time (merged: city and cultural together)
	mergedGroups, ongoingEvents, ongoingCityCount, ongoingPlaza, ongoingNearby, ongoingCityPlaza, ongoingCityNearby := render.GroupMixedEventsByTime(
		filteredCityEvents, filteredEvents, now,
		cfg.Filter.Latitude, cfg.Filter.Longitude, weatherMap)

	// Count events with/without weather
	if buildReport.Weather != nil {
		eventsMatched := 0
		eventsUnmatched := 0
		for _, group := range mergedGroups {
			for _, evt := range group.Events {
				if evt.Weather != nil {
					eventsMatched++
				} else {
					eventsUnmatched++
				}
			}
		}
		for _, evt := range ongoingEvents {
			if evt.Weather != nil {
				eventsMatched++
			} else {
				eventsUnmatched++
			}
		}
		buildReport.Weather.EventsMatched = eventsMatched
		buildReport.Weather.EventsUnmatched = eventsUnmatched
		log.Printf("Weather matching: %d events with weather, %d without", eventsMatched, eventsUnmatched)
	}

	// Convert to JSON format (keep original flat structure for API)
	var culturalJSONEvents []render.JSONEvent
	for _, evt := range filteredEvents {
		culturalJSONEvents = append(culturalJSONEvents, render.JSONEvent{
			ID:         evt.ID,
			Title:      evt.Title,
			StartTime:  evt.StartTime.Format(time.RFC3339),
			EndTime:    evt.EndTime.Format(time.RFC3339), // ADDED: Include end time
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
			EndTime:    evt.EndDate.Format(time.RFC3339), // ADDED: Include end time
			VenueName:  evt.Venue,
			DetailsURL: evt.WebURL,
		})
	}

	// Count total events in merged groups by type
	totalCityEvents := 0
	totalCulturalEvents := 0
	for _, group := range mergedGroups {
		for _, evt := range group.Events {
			if evt.EventType == "city" {
				totalCityEvents++
			} else {
				totalCulturalEvents++
			}
		}
	}
	for _, evt := range ongoingEvents {
		if evt.EventType == "city" {
			totalCityEvents++
		} else {
			totalCulturalEvents++
		}
	}

	// Render outputs
	outDirPath := filepath.Dir(cfg.Output.HTMLPath)
	if err := os.MkdirAll(outDirPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Calculate total distance-filtered counts across all groups
	totalPlaza := ongoingPlaza
	totalCityPlaza := ongoingCityPlaza
	for _, group := range mergedGroups {
		totalPlaza += group.CountPlaza
		totalCityPlaza += group.CityPlaza
	}

	// Render HTML with grouped events
	htmlStart := time.Now()
	htmlRenderer := render.NewHTMLRenderer(*templatePath)
	htmlData := render.GroupedTemplateData{
		Lang:                "es",
		BasePath:            *basePath,
		CSSHash:             readCSSHash(outDirPath),
		LastUpdated:         now.Format("2006-01-02 15:04 MST"),
		GitCommit:           version.GitCommit,
		TotalEvents:         totalCityEvents + totalCulturalEvents,
		TotalCityEvents:     totalCityEvents,
		TotalCulturalEvents: totalCulturalEvents,
		ShowCulturalDefault: true, // Cultural events shown by default
		Groups:              mergedGroups,
		OngoingEvents:       ongoingEvents,
		OngoingCityCount:    ongoingCityCount,
		OngoingPlaza:        ongoingPlaza,
		OngoingNearby:       ongoingNearby,
		OngoingCityPlaza:    ongoingCityPlaza,
		OngoingCityNearby:   ongoingCityNearby,
		TotalPlaza:          totalPlaza,
		TotalNearby:         totalCityEvents + totalCulturalEvents,
		TotalCityPlaza:      totalCityPlaza,
		TotalCityNearby:     totalCityEvents,
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

	// Record final event count (total of both pipelines)
	buildReport.TotalEvents = len(filteredEvents) + len(filteredCityEvents)

	// Export request audit trail
	requestAuditPath := filepath.Join(cfg.Snapshot.DataDir, "request-audit.json")
	if err = client.Auditor().Export(requestAuditPath); err != nil {
		log.Printf("Warning: failed to export audit trail: %v", err)
	} else {
		log.Printf("Request audit exported to: %s", requestAuditPath)
	}

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
