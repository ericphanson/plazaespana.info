package event

import (
	"testing"
	"time"
)

func TestCityEvent_Creation(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	event := CityEvent{
		ID:          "evt-123",
		Title:       "Madrid Gaming Festival",
		Description: "An exciting gaming event in the city center",
		StartDate:   now,
		EndDate:     later,
		Venue:       "Plaza de España",
		Address:     "Plaza de España, 28008 Madrid",
		Latitude:    40.42338,
		Longitude:   -3.71217,
		Category:    "Gaming",
		Subcategory: "eSports",
		WebURL:      "https://example.com/event/123",
		ImageURL:    "https://example.com/images/event-123.jpg",
		Price:       "Free",
	}

	// Verify all fields are set correctly
	if event.ID != "evt-123" {
		t.Errorf("Expected ID 'evt-123', got '%s'", event.ID)
	}
	if event.Title != "Madrid Gaming Festival" {
		t.Errorf("Expected Title 'Madrid Gaming Festival', got '%s'", event.Title)
	}
	if event.Description != "An exciting gaming event in the city center" {
		t.Errorf("Expected specific description, got '%s'", event.Description)
	}
	if !event.StartDate.Equal(now) {
		t.Errorf("Expected StartDate to be %v, got %v", now, event.StartDate)
	}
	if !event.EndDate.Equal(later) {
		t.Errorf("Expected EndDate to be %v, got %v", later, event.EndDate)
	}
	if event.Venue != "Plaza de España" {
		t.Errorf("Expected Venue 'Plaza de España', got '%s'", event.Venue)
	}
	if event.Address != "Plaza de España, 28008 Madrid" {
		t.Errorf("Expected specific address, got '%s'", event.Address)
	}
	if event.Latitude != 40.42338 {
		t.Errorf("Expected Latitude 40.42338, got %f", event.Latitude)
	}
	if event.Longitude != -3.71217 {
		t.Errorf("Expected Longitude -3.71217, got %f", event.Longitude)
	}
	if event.Category != "Gaming" {
		t.Errorf("Expected Category 'Gaming', got '%s'", event.Category)
	}
	if event.Subcategory != "eSports" {
		t.Errorf("Expected Subcategory 'eSports', got '%s'", event.Subcategory)
	}
	if event.WebURL != "https://example.com/event/123" {
		t.Errorf("Expected specific WebURL, got '%s'", event.WebURL)
	}
	if event.ImageURL != "https://example.com/images/event-123.jpg" {
		t.Errorf("Expected specific ImageURL, got '%s'", event.ImageURL)
	}
	if event.Price != "Free" {
		t.Errorf("Expected Price 'Free', got '%s'", event.Price)
	}
}

func TestCityEvent_EventType(t *testing.T) {
	event := CityEvent{
		ID:    "evt-456",
		Title: "Summer Festival",
	}

	eventType := event.EventType()
	expected := "city"

	if eventType != expected {
		t.Errorf("Expected EventType() to return '%s', got '%s'", expected, eventType)
	}
}

func TestCityEvent_Distance(t *testing.T) {
	// Event at Plaza de España
	event := CityEvent{
		ID:        "evt-789",
		Title:     "Event at Plaza de España",
		Latitude:  40.42338,
		Longitude: -3.71217,
	}

	tests := []struct {
		name       string
		lat        float64
		lon        float64
		wantApprox float64
		tolerance  float64
	}{
		{
			name:       "distance to same location",
			lat:        40.42338,
			lon:        -3.71217,
			wantApprox: 0.0,
			tolerance:  0.001,
		},
		{
			name:       "distance to nearby point (~350m north)",
			lat:        40.42650,
			lon:        -3.71217,
			wantApprox: 0.35,
			tolerance:  0.02,
		},
		{
			name:       "distance to far point (~5km north)",
			lat:        40.46838,
			lon:        -3.71217,
			wantApprox: 5.0,
			tolerance:  0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := event.Distance(tt.lat, tt.lon)

			if distance < tt.wantApprox-tt.tolerance || distance > tt.wantApprox+tt.tolerance {
				t.Errorf("Distance(%f, %f) = %f, want approximately %f ± %f",
					tt.lat, tt.lon, distance, tt.wantApprox, tt.tolerance)
			}
		})
	}
}

func TestCityEvent_DistanceWithZeroCoordinates(t *testing.T) {
	// Event with no coordinates
	event := CityEvent{
		ID:        "evt-000",
		Title:     "Event with no coordinates",
		Latitude:  0.0,
		Longitude: 0.0,
	}

	// Distance to Plaza de España should be large (it's in the Gulf of Guinea!)
	distance := event.Distance(40.42338, -3.71217)

	// Should be thousands of km away
	if distance < 4000 {
		t.Errorf("Expected large distance for (0,0) to Madrid, got %f km", distance)
	}
}
