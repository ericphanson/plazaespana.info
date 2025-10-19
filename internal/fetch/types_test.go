package fetch

import (
	"encoding/json"
	"testing"
)

func TestEvent_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"@id": "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json",
		"@context": {
			"@vocab": "http://www.w3.org/ns/dcat#"
		},
		"@graph": [{
			"ID-EVENTO": "12345",
			"TITULO": "Test Event",
			"FECHA": "01/11/2025",
			"FECHA-FIN": "01/11/2025",
			"HORA": "19:00",
			"NOMBRE-INSTALACION": "Test Venue",
			"COORDENADA-LATITUD": 40.42338,
			"COORDENADA-LONGITUD": -3.71217,
			"CONTENT-URL": "https://example.com/event"
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
	if event.IDEvento != "12345" {
		t.Errorf("Expected IDEvento '12345', got '%s'", event.IDEvento)
	}
	if event.Titulo != "Test Event" {
		t.Errorf("Expected Titulo 'Test Event', got '%s'", event.Titulo)
	}
	if event.Lat != 40.42338 {
		t.Errorf("Expected Lat 40.42338, got %f", event.Lat)
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
