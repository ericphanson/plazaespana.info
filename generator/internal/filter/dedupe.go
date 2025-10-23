package filter

import (
	"log"

	"github.com/ericphanson/plazaespana.info/internal/fetch"
)

// DeduplicateByID removes duplicate events based on ID-EVENTO field.
// Keeps the first occurrence of each unique ID.
func DeduplicateByID(events []fetch.RawEvent) []fetch.RawEvent {
	seen := make(map[string]bool)
	var result []fetch.RawEvent

	emptyIDCount := 0
	for i, event := range events {
		if event.IDEvento == "" {
			emptyIDCount++
			// Debug: log first few events with empty ID to see what fields they have
			if i < 3 {
				log.Printf("DEBUG: Event %d has empty IDEvento. Titulo=%q, Fecha=%q, Lat=%.5f, Lon=%.5f",
					i, event.Titulo, event.Fecha, event.Lat, event.Lon)
			}
			continue // Skip events without ID
		}
		if !seen[event.IDEvento] {
			seen[event.IDEvento] = true
			result = append(result, event)
		}
	}

	if emptyIDCount > 0 {
		log.Printf("DEBUG: Skipped %d events with empty IDEvento (total input: %d)", emptyIDCount, len(events))
	}

	return result
}
