package filter

import "github.com/yourusername/madrid-events/internal/fetch"

// DeduplicateByID removes duplicate events based on ID-EVENTO field.
// Keeps the first occurrence of each unique ID.
func DeduplicateByID(events []fetch.RawEvent) []fetch.RawEvent {
	seen := make(map[string]bool)
	var result []fetch.RawEvent

	for _, event := range events {
		if event.IDEvento == "" {
			continue // Skip events without ID
		}
		if !seen[event.IDEvento] {
			seen[event.IDEvento] = true
			result = append(result, event)
		}
	}

	return result
}
