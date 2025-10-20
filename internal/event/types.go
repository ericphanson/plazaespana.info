package event

import "time"

// FilterResult tracks all filter decisions for a single event.
// This structure records the outcome of each filtering stage and
// the final decision about whether the event should be rendered.
type FilterResult struct {
	// Location filtering - distrito
	HasDistrito     bool   `json:"has_distrito"`
	DistritoMatched bool   `json:"distrito_matched"` // if has distrito, did it match target?
	Distrito        string `json:"distrito,omitempty"`

	// Location filtering - GPS
	HasCoordinates bool    `json:"has_coordinates"`
	GPSDistanceKm  float64 `json:"gps_distance_km,omitempty"` // km from reference point
	WithinRadius   bool    `json:"within_radius"`

	// Location filtering - text matching (fallback)
	TextMatched bool `json:"text_matched"`

	// Time filtering
	StartDate time.Time `json:"start_date,omitempty"`
	EndDate   time.Time `json:"end_date,omitempty"`
	DaysOld   int       `json:"days_old"` // days since start (negative = future)
	TooOld    bool      `json:"too_old"`  // started more than cutoff days ago?

	// Final decision
	Kept         bool   `json:"kept"`          // true = will be rendered, false = filtered out
	FilterReason string `json:"filter_reason"` // human-readable: "outside distrito", "too old", "kept", etc.
}
