package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yourusername/madrid-events/internal/fetch"
	"github.com/yourusername/madrid-events/internal/filter"
	"github.com/yourusername/madrid-events/internal/render"
	"github.com/yourusername/madrid-events/internal/snapshot"
)

func main() {
	// Parse flags
	jsonURL := flag.String("json-url", "", "Madrid events JSON URL")
	xmlURL := flag.String("xml-url", "", "Madrid events XML URL (fallback)")
	csvURL := flag.String("csv-url", "", "Madrid events CSV URL (fallback)")
	outDir := flag.String("out-dir", "./public", "Output directory for static files")
	dataDir := flag.String("data-dir", "./data", "Data directory for snapshots")
	lat := flag.Float64("lat", 40.42338, "Reference latitude (Plaza de España)")
	lon := flag.Float64("lon", -3.71217, "Reference longitude (Plaza de España)")
	radiusKm := flag.Float64("radius-km", 0.35, "Filter radius in kilometers")
	timezone := flag.String("timezone", "Europe/Madrid", "Timezone for event times")

	flag.Parse()

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

	log.Println("Fetching JSON from:", *jsonURL)
	jsonResp, err := client.FetchJSON(*jsonURL)
	if err == nil && jsonResp != nil {
		rawEvents = jsonResp.Graph
		log.Printf("Fetched %d events from JSON", len(rawEvents))
	} else {
		fetchErr = err
		log.Printf("JSON fetch failed: %v", err)

		if *xmlURL != "" {
			log.Println("Falling back to XML:", *xmlURL)
			rawEvents, err = client.FetchXML(*xmlURL)
			if err == nil {
				fetchErr = nil
				log.Printf("Fetched %d events from XML", len(rawEvents))
			} else {
				log.Printf("XML fetch failed: %v", err)
			}
		}

		if fetchErr != nil && *csvURL != "" {
			log.Println("Falling back to CSV:", *csvURL)
			rawEvents, err = client.FetchCSV(*csvURL)
			if err == nil {
				fetchErr = nil
				log.Printf("Fetched %d events from CSV", len(rawEvents))
			} else {
				log.Printf("CSV fetch failed: %v", err)
			}
		}
	}

	// If all fetches failed, try loading snapshot
	if fetchErr != nil {
		log.Println("All fetch attempts failed, loading snapshot...")
		rawEvents, err = snapMgr.LoadSnapshot()
		if err != nil {
			log.Fatalf("Failed to load snapshot: %v", err)
		}
		log.Printf("Loaded %d events from snapshot (stale data)", len(rawEvents))
	} else {
		// Save successful fetch to snapshot
		if err := snapMgr.SaveSnapshot(rawEvents); err != nil {
			log.Printf("Warning: failed to save snapshot: %v", err)
		}
	}

	// Deduplicate
	rawEvents = filter.DeduplicateByID(rawEvents)
	log.Printf("After deduplication: %d events", len(rawEvents))

	// Filter by location and time
	now := time.Now().In(loc)
	var filteredEvents []fetch.RawEvent

	for _, event := range rawEvents {
		// Skip if missing coordinates
		if event.Lat == 0 || event.Lon == 0 {
			continue
		}

		// Check geographic proximity
		if !filter.WithinRadius(*lat, *lon, event.Lat, event.Lon, *radiusKm) {
			continue
		}

		// Parse and check if event is in the future
		startTime, err := filter.ParseEventDateTime(event.Fecha, event.Hora, loc)
		if err != nil {
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
			continue
		}

		filteredEvents = append(filteredEvents, event)
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
	htmlRenderer := render.NewHTMLRenderer("templates/index.tmpl.html")
	htmlData := render.TemplateData{
		Lang:        "es",
		CSSHash:     "placeholder",
		LastUpdated: now.Format("2006-01-02 15:04 MST"),
		Events:      templateEvents,
	}
	htmlPath := fmt.Sprintf("%s/index.html", *outDir)
	if err := htmlRenderer.Render(htmlData, htmlPath); err != nil {
		log.Fatalf("Failed to render HTML: %v", err)
	}
	log.Println("Generated:", htmlPath)

	// Render JSON
	jsonRenderer := render.NewJSONRenderer()
	jsonPath := fmt.Sprintf("%s/events.json", *outDir)
	if err := jsonRenderer.Render(jsonEvents, jsonPath); err != nil {
		log.Fatalf("Failed to render JSON: %v", err)
	}
	log.Println("Generated:", jsonPath)

	log.Println("Build complete!")
}
