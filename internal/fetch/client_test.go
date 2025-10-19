package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient(5 * time.Second)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.httpClient == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}

func TestClient_FetchWithUserAgent(t *testing.T) {
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"@graph":[]}`))
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)
	_, err := client.FetchJSON(server.URL)
	if err != nil {
		t.Fatalf("FetchJSON failed: %v", err)
	}

	if capturedUserAgent == "" {
		t.Error("User-Agent header not set")
	}
	if capturedUserAgent != "madrid-events-site-generator/1.0 (https://github.com/ericphanson/madrid-events)" {
		t.Errorf("Unexpected User-Agent: %s", capturedUserAgent)
	}
}

func TestClient_FetchXML(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<response>
	<event>
		<ID-EVENTO>XML-001</ID-EVENTO>
		<TITULO>XML Event</TITULO>
		<FECHA>20/11/2025</FECHA>
		<FECHA-FIN>20/11/2025</FECHA-FIN>
		<HORA>18:00</HORA>
		<NOMBRE-INSTALACION>Test Venue</NOMBRE-INSTALACION>
		<COORDENADA-LATITUD>40.42</COORDENADA-LATITUD>
		<COORDENADA-LONGITUD>-3.71</COORDENADA-LONGITUD>
	</event>
</response>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(xmlData))
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)
	events, err := client.FetchXML(server.URL)
	if err != nil {
		t.Fatalf("FetchXML failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].IDEvento != "XML-001" {
		t.Errorf("Expected IDEvento 'XML-001', got '%s'", events[0].IDEvento)
	}
}

func TestClient_FetchCSV_Semicolon(t *testing.T) {
	csvData := `ID-EVENTO;TITULO;FECHA;FECHA-FIN;HORA;NOMBRE-INSTALACION;COORDENADA-LATITUD;COORDENADA-LONGITUD
CSV-001;CSV Event;25/11/2025;25/11/2025;17:30;CSV Venue;40.423;-3.712`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(csvData))
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)
	events, err := client.FetchCSV(server.URL)
	if err != nil {
		t.Fatalf("FetchCSV failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].IDEvento != "CSV-001" {
		t.Errorf("Expected IDEvento 'CSV-001', got '%s'", events[0].IDEvento)
	}
	if events[0].Titulo != "CSV Event" {
		t.Errorf("Expected Titulo 'CSV Event', got '%s'", events[0].Titulo)
	}
	// Validate coordinates
	if events[0].Lat != 40.42 && events[0].Lat != 40.423 {
		t.Errorf("Expected Lat ~40.42, got %f", events[0].Lat)
	}
	// Check longitude is negative and close to expected value
	if events[0].Lon >= 0 || events[0].Lon < -4 || events[0].Lon > -3 {
		t.Errorf("Expected Lon ~-3.71, got %f", events[0].Lon)
	}
}

func TestClient_FetchCSV_Comma(t *testing.T) {
	csvData := `ID-EVENTO,TITULO,FECHA,FECHA-FIN,HORA,NOMBRE-INSTALACION,COORDENADA-LATITUD,COORDENADA-LONGITUD
CSV-002,CSV Event 2,26/11/2025,26/11/2025,18:00,Venue 2,40.42,-3.71`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(csvData))
	}))
	defer server.Close()

	client := NewClient(5 * time.Second)
	events, err := client.FetchCSV(server.URL)
	if err != nil {
		t.Fatalf("FetchCSV failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].IDEvento != "CSV-002" {
		t.Errorf("Expected IDEvento 'CSV-002', got '%s'", events[0].IDEvento)
	}
	// Validate coordinates
	if events[0].Lat != 40.42 && events[0].Lat != 40.423 {
		t.Errorf("Expected Lat ~40.42, got %f", events[0].Lat)
	}
	// Check longitude is negative and close to expected value
	if events[0].Lon >= 0 || events[0].Lon < -4 || events[0].Lon > -3 {
		t.Errorf("Expected Lon ~-3.71, got %f", events[0].Lon)
	}
}
