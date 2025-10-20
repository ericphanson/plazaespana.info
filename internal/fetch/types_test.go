package fetch

import (
	"encoding/json"
	"testing"
)

func TestEvent_UnmarshalJSON(t *testing.T) {
	// Test with JSON-LD format (actual Madrid API format)
	jsonData := `{
		"@id": "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
		"@context": {
			"@vocab": "http://www.w3.org/ns/dcat#"
		},
		"@graph": [{
			"id": "12345",
			"title": "Test Event",
			"description": "Test description",
			"dtstart": "2025-11-01 19:00:00.0",
			"dtend": "2025-11-01 22:00:00.0",
			"event-location": "Test Venue",
			"latitude": 40.42338,
			"longitude": -3.71217,
			"link": "https://example.com/event"
		}]
	}`

	var response JSONResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(response.Graph) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(response.Graph))
	}

	event := response.Graph[0]
	if event.ID != "12345" {
		t.Errorf("Expected ID '12345', got '%s'", event.ID)
	}
	if event.Title != "Test Event" {
		t.Errorf("Expected Title 'Test Event', got '%s'", event.Title)
	}
	if event.Latitude != 40.42338 {
		t.Errorf("Expected Latitude 40.42338, got %f", event.Latitude)
	}
	if event.Longitude != -3.71217 {
		t.Errorf("Expected Longitude -3.71217, got %f", event.Longitude)
	}
	if event.Location != "Test Venue" {
		t.Errorf("Expected Location 'Test Venue', got '%s'", event.Location)
	}
}

func TestRawEvent_Fields(t *testing.T) {
	event := RawEvent{
		IDEvento:          "TEST-001",
		Titulo:            "Concert",
		Fecha:             "15/11/2025",
		FechaFin:          "15/11/2025",
		Hora:              "20:00",
		NombreInstalacion: "Plaza de España",
		Lat:               40.42338,
		Lon:               -3.71217,
		ContentURL:        "https://madrid.es/event/001",
	}

	if event.IDEvento != "TEST-001" {
		t.Errorf("IDEvento mismatch")
	}
	if event.NombreInstalacion != "Plaza de España" {
		t.Errorf("NombreInstalacion mismatch")
	}
}
