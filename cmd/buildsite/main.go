package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ericphanson/madrid-events/internal/fetch"
	"github.com/ericphanson/madrid-events/internal/filter"
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

		// Write markdown report
		mdPath := filepath.Join(outputDir, "build-report.md")
		if f, err := os.Create(mdPath); err == nil {
			buildReport.WriteMarkdown(f)
			f.Close()
			log.Println("Build report written to:", mdPath)
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

	// Load timezone
	loc, err := time.LoadLocation(*timezone)
	if err != nil {
		log.Fatalf("Invalid timezone: %v", err)
	}

	// Initialize components
	client := fetch.NewClient(30 * time.Second)
	snapMgr := snapshot.NewManager(*dataDir)

	// Fetch events (with fallback chain)
	var rawEvents []fetch.RawEvent
	var fetchErr error
	fetchStart := time.Now()

	// Try JSON
	log.Println("Fetching JSON from:", *jsonURL)
	jsonStart := time.Now()
	jsonResult := client.FetchJSON(*jsonURL, loc)
	jsonDuration := time.Since(jsonStart)

	jsonAttempt := report.FetchAttempt{
		Source:   "JSON",
		URL:      *jsonURL,
		Duration: jsonDuration,
	}

	if len(jsonResult.Events) > 0 {
		// Convert CanonicalEvents back to RawEvents (temporary until Task 7)
		for _, sourced := range jsonResult.Events {
			evt := sourced.Event
			rawEvents = append(rawEvents, fetch.RawEvent{
				IDEvento:          evt.ID,
				Titulo:            evt.Title,
				Descripcion:       evt.Description,
				Fecha:             evt.StartTime.Format("2006-01-02"),
				Hora:              evt.StartTime.Format("15:04"),
				NombreInstalacion: evt.VenueName,
				ContentURL:        evt.DetailsURL,
				Lat:               evt.Latitude,
				Lon:               evt.Longitude,
			})
		}
		jsonAttempt.Status = "SUCCESS"
		jsonAttempt.EventCount = len(rawEvents)
		jsonAttempt.HTTPStatus = 200
		buildReport.Fetching.SourceUsed = "JSON"
		log.Printf("Fetched %d events from JSON", len(rawEvents))
	} else {
		fetchErr = fmt.Errorf("no events parsed from JSON")
		jsonAttempt.Status = "FAILED"
		if len(jsonResult.Errors) > 0 {
			jsonAttempt.Error = jsonResult.Errors[0].Error.Error()
		} else {
			jsonAttempt.Error = "no events parsed"
		}
		log.Printf("JSON fetch failed: %v", fetchErr)

		// Try XML
		if *xmlURL != "" {
			log.Println("Falling back to XML:", *xmlURL)
			xmlStart := time.Now()
			xmlResult := client.FetchXML(*xmlURL, loc)
			xmlDuration := time.Since(xmlStart)

			xmlAttempt := report.FetchAttempt{
				Source:   "XML",
				URL:      *xmlURL,
				Duration: xmlDuration,
			}

			if len(xmlResult.Events) > 0 {
				fetchErr = nil
				rawEvents = nil // Clear JSON events
				for _, sourced := range xmlResult.Events {
					evt := sourced.Event
					rawEvents = append(rawEvents, fetch.RawEvent{
						IDEvento:          evt.ID,
						Titulo:            evt.Title,
						Descripcion:       evt.Description,
						Fecha:             evt.StartTime.Format("2006-01-02"),
						Hora:              evt.StartTime.Format("15:04"),
						NombreInstalacion: evt.VenueName,
						ContentURL:        evt.DetailsURL,
						Lat:               evt.Latitude,
						Lon:               evt.Longitude,
					})
				}
				xmlAttempt.Status = "SUCCESS"
				xmlAttempt.EventCount = len(rawEvents)
				xmlAttempt.HTTPStatus = 200
				buildReport.Fetching.SourceUsed = "XML"
				log.Printf("Fetched %d events from XML", len(rawEvents))
			} else {
				xmlAttempt.Status = "FAILED"
				if len(xmlResult.Errors) > 0 {
					xmlAttempt.Error = xmlResult.Errors[0].Error.Error()
				} else {
					xmlAttempt.Error = "no events parsed"
				}
				log.Printf("XML fetch failed: no events parsed")
			}
			buildReport.Fetching.Attempts = append(buildReport.Fetching.Attempts, xmlAttempt)
		}

		// Try CSV
		if fetchErr != nil && *csvURL != "" {
			log.Println("Falling back to CSV:", *csvURL)
			csvStart := time.Now()
			csvResult := client.FetchCSV(*csvURL, loc)
			csvDuration := time.Since(csvStart)

			csvAttempt := report.FetchAttempt{
				Source:   "CSV",
				URL:      *csvURL,
				Duration: csvDuration,
			}

			if len(csvResult.Events) > 0 {
				fetchErr = nil
				rawEvents = nil // Clear previous events
				for _, sourced := range csvResult.Events {
					evt := sourced.Event
					rawEvents = append(rawEvents, fetch.RawEvent{
						IDEvento:          evt.ID,
						Titulo:            evt.Title,
						Descripcion:       evt.Description,
						Fecha:             evt.StartTime.Format("2006-01-02"),
						Hora:              evt.StartTime.Format("15:04"),
						NombreInstalacion: evt.VenueName,
						ContentURL:        evt.DetailsURL,
						Lat:               evt.Latitude,
						Lon:               evt.Longitude,
					})
				}
				csvAttempt.Status = "SUCCESS"
				csvAttempt.EventCount = len(rawEvents)
				csvAttempt.HTTPStatus = 200
				buildReport.Fetching.SourceUsed = "CSV"
				log.Printf("Fetched %d events from CSV", len(rawEvents))
			} else {
				csvAttempt.Status = "FAILED"
				if len(csvResult.Errors) > 0 {
					csvAttempt.Error = csvResult.Errors[0].Error.Error()
				} else {
					csvAttempt.Error = "no events parsed"
				}
				log.Printf("CSV fetch failed: no events parsed")
			}
			buildReport.Fetching.Attempts = append(buildReport.Fetching.Attempts, csvAttempt)
		}
	}
	buildReport.Fetching.Attempts = append([]report.FetchAttempt{jsonAttempt}, buildReport.Fetching.Attempts...)

	// If all fetches failed, try loading snapshot
	if fetchErr != nil {
		log.Println("All fetch attempts failed, loading snapshot...")
		rawEvents, err = snapMgr.LoadSnapshot()
		if err != nil {
			buildReport.ExitStatus = "FAILED"
			log.Fatalf("Failed to load snapshot: %v", err)
		}
		buildReport.Fetching.SourceUsed = "SNAPSHOT"
		buildReport.AddWarning("Using stale snapshot data - all fetch attempts failed")
		log.Printf("Loaded %d events from snapshot (stale data)", len(rawEvents))
	} else {
		// Save successful fetch to snapshot
		if err := snapMgr.SaveSnapshot(rawEvents); err != nil {
			log.Printf("Warning: failed to save snapshot: %v", err)
		}
	}

	buildReport.Fetching.TotalDuration = time.Since(fetchStart)

	// Deduplicate
	dedupStart := time.Now()
	inputCount := len(rawEvents)
	rawEvents = filter.DeduplicateByID(rawEvents)
	outputCount := len(rawEvents)
	buildReport.Processing.Deduplication = report.DeduplicationStats{
		Input:      inputCount,
		Duplicates: inputCount - outputCount,
		Output:     outputCount,
		Duration:   time.Since(dedupStart),
	}
	log.Printf("After deduplication: %d events", len(rawEvents))

	// Filter by location and time
	now := time.Now().In(loc)
	geoStart := time.Now()
	var filteredEvents []fetch.RawEvent

	missingCoords := 0
	outsideRadius := 0
	parseFailures := 0
	pastEvents := 0

	for _, event := range rawEvents {
		// Skip if missing coordinates
		if event.Lat == 0 || event.Lon == 0 {
			missingCoords++
			continue
		}

		// Check geographic proximity
		if !filter.WithinRadius(*lat, *lon, event.Lat, event.Lon, *radiusKm) {
			outsideRadius++
			continue
		}

		// Parse and check if event is in the future
		startTime, err := filter.ParseEventDateTime(event.Fecha, event.Hora, loc)
		if err != nil {
			parseFailures++
			log.Printf("Skipping event %s (invalid date): %v", event.IDEvento, err)
			continue
		}

		// Use end date if available, otherwise use start date
		endDate := event.FechaFin
		if endDate == "" {
			endDate = event.Fecha
		}
		endTime, err := filter.ParseEventDateTime(endDate, "", loc)
		if err != nil {
			endTime = startTime
		}

		if !filter.IsInFuture(endTime, now) {
			pastEvents++
			continue
		}

		filteredEvents = append(filteredEvents, event)
	}

	// Record geo filter stats
	geoDuration := time.Since(geoStart)
	buildReport.Processing.GeoFilter = report.GeoFilterStats{
		RefLat:        *lat,
		RefLon:        *lon,
		Radius:        *radiusKm,
		Input:         len(rawEvents),
		MissingCoords: missingCoords,
		OutsideRadius: outsideRadius,
		Kept:          len(filteredEvents) + pastEvents + parseFailures, // Events that passed geo filter
		Duration:      geoDuration,
	}

	// Record time filter stats
	buildReport.Processing.TimeFilter = report.TimeFilterStats{
		ReferenceTime: now,
		Timezone:      *timezone,
		Input:         len(filteredEvents) + pastEvents + parseFailures,
		ParseFailures: parseFailures,
		PastEvents:    pastEvents,
		Kept:          len(filteredEvents),
		Duration:      0, // Included in geo filter duration
	}

	// Add warnings if needed
	if len(filteredEvents) < len(rawEvents)/100 { // Less than 1%
		buildReport.AddWarning("Geographic radius very restrictive (%.2fkm) - only %.1f%% of events kept",
			*radiusKm, float64(len(filteredEvents))*100/float64(len(rawEvents)))
		buildReport.AddRecommendation("Consider increasing -radius-km to 1.0-2.0 for better coverage")
	}

	log.Printf("After filtering: %d events", len(filteredEvents))

	// Convert to template format
	var templateEvents []render.TemplateEvent
	var jsonEvents []render.JSONEvent

	for _, event := range filteredEvents {
		startTime, _ := filter.ParseEventDateTime(event.Fecha, event.Hora, loc)

		templateEvents = append(templateEvents, render.TemplateEvent{
			IDEvento:          event.IDEvento,
			Titulo:            event.Titulo,
			StartHuman:        startTime.Format("02/01/2006 15:04"),
			NombreInstalacion: event.NombreInstalacion,
			ContentURL:        event.ContentURL,
		})

		jsonEvents = append(jsonEvents, render.JSONEvent{
			ID:         event.IDEvento,
			Title:      event.Titulo,
			StartTime:  startTime.Format(time.RFC3339),
			VenueName:  event.NombreInstalacion,
			DetailsURL: event.ContentURL,
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
	jsonStart = time.Now()
	jsonRenderer := render.NewJSONRenderer()
	jsonPath := fmt.Sprintf("%s/events.json", *outDir)
	jsonErr := jsonRenderer.Render(jsonEvents, jsonPath)
	jsonDuration = time.Since(jsonStart)

	if jsonErr != nil {
		buildReport.Output.JSON = report.OutputFile{
			Path:     jsonPath,
			Status:   "FAILED",
			Error:    jsonErr.Error(),
			Duration: jsonDuration,
		}
		log.Fatalf("Failed to render JSON: %v", jsonErr)
	}

	jsonInfo, _ := os.Stat(jsonPath)
	buildReport.Output.JSON = report.OutputFile{
		Path:     jsonPath,
		Size:     jsonInfo.Size(),
		Status:   "SUCCESS",
		Duration: jsonDuration,
	}
	log.Println("Generated:", jsonPath)

	// Record final event count
	buildReport.EventsCount = len(filteredEvents)

	log.Println("Build complete!")
}
