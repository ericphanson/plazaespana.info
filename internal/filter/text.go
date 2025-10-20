package filter

import "strings"

// MatchesLocation checks if event text mentions target location.
// Used as fallback when coordinates are missing.
func MatchesLocation(venueName, address, description string, keywords []string) bool {
	// Combine all text fields
	text := strings.ToLower(venueName + " " + address + " " + description)

	// Check if any keyword appears
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}
