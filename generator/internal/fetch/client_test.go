package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.httpClient == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.httpClient.Timeout)
	}
	if client.cache == nil {
		t.Error("Expected non-nil cache")
	}
	if client.throttle == nil {
		t.Error("Expected non-nil throttle")
	}
	if client.auditor == nil {
		t.Error("Expected non-nil auditor")
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

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	result := client.FetchJSON(server.URL, loc)
	if len(result.Errors) > 0 {
		t.Fatalf("FetchJSON failed: %v", result.Errors[0].Error)
	}

	if capturedUserAgent == "" {
		t.Error("User-Agent header not set")
	}
	if capturedUserAgent != "plazaespana-info-site-generator/1.0 (https://github.com/ericphanson/plazaespana.info)" {
		t.Errorf("Unexpected User-Agent: %s", capturedUserAgent)
	}
}

func TestClient_FetchXML(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Contenidos>
	<contenido>
		<atributos idioma="es">
			<atributo nombre="ID-EVENTO">XML-001</atributo>
			<atributo nombre="TITULO">XML Event</atributo>
			<atributo nombre="FECHA-EVENTO">2025-11-20 00:00:00.0</atributo>
			<atributo nombre="FECHA-FIN-EVENTO">2025-11-20 23:59:00.0</atributo>
			<atributo nombre="HORA-EVENTO">18:00</atributo>
			<atributo nombre="CONTENT-URL">http://example.com</atributo>
			<atributo nombre="LOCALIZACION">
				<atributo nombre="NOMBRE-INSTALACION">Test Venue</atributo>
				<atributo nombre="LATITUD">40.42</atributo>
				<atributo nombre="LONGITUD">-3.71</atributo>
			</atributo>
		</atributos>
	</contenido>
</Contenidos>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(xmlData))
	}))
	defer server.Close()

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	result := client.FetchXML(server.URL, loc)

	if len(result.Errors) > 0 {
		t.Fatalf("FetchXML had errors: %v", result.Errors[0].Error)
	}

	if len(result.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Event.ID != "XML-001" {
		t.Errorf("Expected ID 'XML-001', got '%s'", result.Events[0].Event.ID)
	}
}

func TestClient_FetchCSV_Semicolon(t *testing.T) {
	csvData := `ID-EVENTO;TITULO;FECHA;FECHA-FIN;HORA;NOMBRE-INSTALACION;LATITUD;LONGITUD
CSV-001;CSV Event;2025-11-25 00:00:00.0;2025-11-25 23:59:00.0;17:30;CSV Venue;40.423;-3.712`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(csvData))
	}))
	defer server.Close()

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	result := client.FetchCSV(server.URL, loc)

	if len(result.Errors) > 0 {
		t.Fatalf("FetchCSV had errors: %v", result.Errors[0].Error)
	}

	if len(result.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Event.ID != "CSV-001" {
		t.Errorf("Expected ID 'CSV-001', got '%s'", result.Events[0].Event.ID)
	}
	if result.Events[0].Event.Title != "CSV Event" {
		t.Errorf("Expected Title 'CSV Event', got '%s'", result.Events[0].Event.Title)
	}
	// Validate coordinates
	if result.Events[0].Event.Latitude != 40.423 {
		t.Errorf("Expected Latitude 40.423, got %f", result.Events[0].Event.Latitude)
	}
	if result.Events[0].Event.Longitude != -3.712 {
		t.Errorf("Expected Longitude -3.712, got %f", result.Events[0].Event.Longitude)
	}
}

func TestClient_FetchCSV_Comma(t *testing.T) {
	csvData := `ID-EVENTO,TITULO,FECHA,FECHA-FIN,HORA,NOMBRE-INSTALACION,LATITUD,LONGITUD
CSV-002,CSV Event 2,2025-11-26 00:00:00.0,2025-11-26 23:59:00.0,18:00,Venue 2,40.42,-3.71`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(csvData))
	}))
	defer server.Close()

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("loading timezone: %v", err)
	}

	config := DefaultDevelopmentConfig()
	client, err := NewClient(5*time.Second, config, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	result := client.FetchCSV(server.URL, loc)

	if len(result.Errors) > 0 {
		t.Fatalf("FetchCSV had errors: %v", result.Errors[0].Error)
	}

	if len(result.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Event.ID != "CSV-002" {
		t.Errorf("Expected ID 'CSV-002', got '%s'", result.Events[0].Event.ID)
	}
	// Validate coordinates
	if result.Events[0].Event.Latitude != 40.42 {
		t.Errorf("Expected Latitude 40.42, got %f", result.Events[0].Event.Latitude)
	}
	if result.Events[0].Event.Longitude != -3.71 {
		t.Errorf("Expected Longitude -3.71, got %f", result.Events[0].Event.Longitude)
	}
}
