package fetch

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestParseEsmadridXML tests parsing of a single ESMadrid service element
func TestParseEsmadridXML(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<serviceList>
	<service fechaActualizacion="2025-10-19" id="107777">
		<basicData>
			<language>es</language>
			<name><![CDATA[Slamming]]></name>
			<title><![CDATA[Slamming]]></title>
			<body><![CDATA[<p>Performance inspired by punk concert behavior.</p>]]></body>
			<web>https://www.esmadrid.com/agenda/slamming-replika-teatro</web>
			<idrt>78001</idrt>
			<nombrert>Réplika Teatro</nombrert>
		</basicData>
		<geoData>
			<address>de la Explanada, 14</address>
			<zipcode>28040</zipcode>
			<locality/>
			<country>Spain</country>
			<latitude>40.448271800000</latitude>
			<longitude>-3.711678700000</longitude>
			<subAdministrativeArea>Madrid</subAdministrativeArea>
		</geoData>
		<multimedia>
			<media type="image">
				<url>https://estaticos.esmadrid.com/example.jpg</url>
			</media>
		</multimedia>
		<extradata>
			<item name="idTipo">6</item>
			<item name="Tipo">Eventos</item>
			<categorias>
				<categoria>
					<item name="idCategoria">6486</item>
					<item name="Categoria">Teatro y danza</item>
					<subcategorias>
						<subcategoria>
							<item name="idSubCategoria">6490</item>
							<item name="SubCategoria">Danza moderna</item>
						</subcategoria>
					</subcategorias>
				</categoria>
			</categorias>
			<item name="Servicios de pago"><![CDATA[<p>General: 18,36 €</p>]]></item>
			<fechas>
				<rango>
					<inicio>26/10/2025</inicio>
					<dias>7</dias>
					<fin>26/10/2025</fin>
				</rango>
			</fechas>
		</extradata>
	</service>
</serviceList>`

	var serviceList EsmadridServiceList
	err := xml.Unmarshal([]byte(xmlData), &serviceList)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(serviceList.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(serviceList.Services))
	}

	svc := serviceList.Services[0]

	// Test basic attributes
	if svc.ID != "107777" {
		t.Errorf("Expected ID '107777', got '%s'", svc.ID)
	}
	if svc.UpdateDate != "2025-10-19" {
		t.Errorf("Expected UpdateDate '2025-10-19', got '%s'", svc.UpdateDate)
	}

	// Test basic data
	if svc.Name != "Slamming" {
		t.Errorf("Expected Name 'Slamming', got '%s'", svc.Name)
	}
	if svc.Title != "Slamming" {
		t.Errorf("Expected Title 'Slamming', got '%s'", svc.Title)
	}
	if svc.Web != "https://www.esmadrid.com/agenda/slamming-replika-teatro" {
		t.Errorf("Expected Web URL, got '%s'", svc.Web)
	}
	if svc.VenueID != "78001" {
		t.Errorf("Expected VenueID '78001', got '%s'", svc.VenueID)
	}
	if svc.VenueName != "Réplika Teatro" {
		t.Errorf("Expected VenueName 'Réplika Teatro', got '%s'", svc.VenueName)
	}

	// Test geo data
	if svc.Address != "de la Explanada, 14" {
		t.Errorf("Expected Address 'de la Explanada, 14', got '%s'", svc.Address)
	}
	if svc.Latitude != "40.448271800000" {
		t.Errorf("Expected Latitude '40.448271800000', got '%s'", svc.Latitude)
	}
	if svc.Longitude != "-3.711678700000" {
		t.Errorf("Expected Longitude '-3.711678700000', got '%s'", svc.Longitude)
	}

	// Test multimedia
	if svc.ImageURL != "https://estaticos.esmadrid.com/example.jpg" {
		t.Errorf("Expected ImageURL, got '%s'", svc.ImageURL)
	}

	// Test extradata extraction
	if svc.Category != "Teatro y danza" {
		t.Errorf("Expected Category 'Teatro y danza', got '%s'", svc.Category)
	}
	if svc.Subcategory != "Danza moderna" {
		t.Errorf("Expected Subcategory 'Danza moderna', got '%s'", svc.Subcategory)
	}
	if svc.StartDate != "26/10/2025" {
		t.Errorf("Expected StartDate '26/10/2025', got '%s'", svc.StartDate)
	}
	if svc.EndDate != "26/10/2025" {
		t.Errorf("Expected EndDate '26/10/2025', got '%s'", svc.EndDate)
	}
}

// TestToCityEvent tests conversion from EsmadridService to CityEvent
func TestToCityEvent(t *testing.T) {
	svc := EsmadridService{
		ID:          "107777",
		UpdateDate:  "2025-10-19",
		Name:        "Slamming",
		Title:       "Slamming",
		Body:        "<p>Performance inspired by punk concert behavior.</p>",
		Web:         "https://www.esmadrid.com/agenda/slamming-replika-teatro",
		VenueID:     "78001",
		VenueName:   "Réplika Teatro",
		Address:     "de la Explanada, 14",
		Latitude:    "40.448271800000",
		Longitude:   "-3.711678700000",
		ImageURL:    "https://estaticos.esmadrid.com/example.jpg",
		Category:    "Teatro y danza",
		Subcategory: "Danza moderna",
		Price:       "<p>General: 18,36 €</p>",
		StartDate:   "26/10/2025",
		EndDate:     "26/10/2025",
	}

	evt, err := svc.ToCityEvent()
	if err != nil {
		t.Fatalf("ToCityEvent failed: %v", err)
	}

	// Check basic fields
	if evt.ID != "107777" {
		t.Errorf("Expected ID '107777', got '%s'", evt.ID)
	}
	if evt.Title != "Slamming" {
		t.Errorf("Expected Title 'Slamming', got '%s'", evt.Title)
	}
	if evt.Venue != "Réplika Teatro" {
		t.Errorf("Expected Venue 'Réplika Teatro', got '%s'", evt.Venue)
	}
	if evt.Address != "de la Explanada, 14" {
		t.Errorf("Expected Address 'de la Explanada, 14', got '%s'", evt.Address)
	}
	if evt.Category != "Teatro y danza" {
		t.Errorf("Expected Category 'Teatro y danza', got '%s'", evt.Category)
	}
	if evt.Subcategory != "Danza moderna" {
		t.Errorf("Expected Subcategory 'Danza moderna', got '%s'", evt.Subcategory)
	}
	if evt.WebURL != "https://www.esmadrid.com/agenda/slamming-replika-teatro" {
		t.Errorf("Expected WebURL, got '%s'", evt.WebURL)
	}
	if evt.ImageURL != "https://estaticos.esmadrid.com/example.jpg" {
		t.Errorf("Expected ImageURL, got '%s'", evt.ImageURL)
	}
	if evt.Price != "<p>General: 18,36 €</p>" {
		t.Errorf("Expected Price, got '%s'", evt.Price)
	}

	// Check coordinates (should be parsed as float64)
	if evt.Latitude != 40.448271800000 {
		t.Errorf("Expected Latitude 40.448271800000, got %f", evt.Latitude)
	}
	if evt.Longitude != -3.711678700000 {
		t.Errorf("Expected Longitude -3.711678700000, got %f", evt.Longitude)
	}

	// Check dates (should be parsed to Europe/Madrid timezone)
	expectedStart := time.Date(2025, 10, 26, 0, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	if !evt.StartDate.Equal(expectedStart) {
		t.Errorf("Expected StartDate %v, got %v", expectedStart, evt.StartDate)
	}
	if !evt.EndDate.Equal(expectedStart) {
		t.Errorf("Expected EndDate %v, got %v", expectedStart, evt.EndDate)
	}
}

// TestToCityEventMissingFields tests handling of missing optional fields
func TestToCityEventMissingFields(t *testing.T) {
	svc := EsmadridService{
		ID:        "123",
		Title:     "Test Event",
		StartDate: "26/10/2025",
		EndDate:   "26/10/2025",
		// Missing: coordinates, venue, category, etc.
	}

	evt, err := svc.ToCityEvent()
	if err != nil {
		t.Fatalf("ToCityEvent should handle missing fields gracefully: %v", err)
	}

	// Should have ID and title at minimum
	if evt.ID != "123" {
		t.Errorf("Expected ID '123', got '%s'", evt.ID)
	}
	if evt.Title != "Test Event" {
		t.Errorf("Expected Title 'Test Event', got '%s'", evt.Title)
	}

	// Missing coordinates should default to 0
	if evt.Latitude != 0.0 {
		t.Errorf("Expected default Latitude 0.0, got %f", evt.Latitude)
	}
	if evt.Longitude != 0.0 {
		t.Errorf("Expected default Longitude 0.0, got %f", evt.Longitude)
	}
}

// TestParseFullFixture tests parsing the complete esmadrid-agenda.xml fixture
func TestParseFullFixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/esmadrid-agenda.xml")
	if err != nil {
		t.Skipf("Skipping full fixture test: %v", err)
		return
	}

	var serviceList EsmadridServiceList
	err = xml.Unmarshal(data, &serviceList)
	if err != nil {
		t.Fatalf("Failed to parse full fixture: %v", err)
	}

	t.Logf("Successfully parsed %d services from fixture", len(serviceList.Services))

	if len(serviceList.Services) == 0 {
		t.Error("Expected at least 1 service in fixture")
	}

	// Test that we can convert all services to CityEvents
	successCount := 0
	for _, svc := range serviceList.Services {
		evt, err := svc.ToCityEvent()
		if err != nil {
			t.Logf("Warning: Failed to convert service %s: %v", svc.ID, err)
			continue
		}
		successCount++

		// Verify minimum required fields are present
		if evt.ID == "" {
			t.Errorf("Service %s converted to event with empty ID", svc.ID)
		}
		if evt.Title == "" {
			t.Errorf("Service %s converted to event with empty Title", svc.ID)
		}
	}

	t.Logf("Successfully converted %d/%d services to CityEvents", successCount, len(serviceList.Services))

	if successCount == 0 {
		t.Error("Expected at least some successful conversions")
	}
}

// TestFetchEsmadridEvents_Success tests successful HTTP fetch and parse
func TestFetchEsmadridEvents_Success(t *testing.T) {
	// Mock ESMadrid XML response
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<serviceList>
	<service id="12345" fechaActualizacion="2025-10-20">
		<basicData>
			<name><![CDATA[Exposición de Arte]]></name>
			<title><![CDATA[Arte Contemporáneo]]></title>
			<body><![CDATA[Una exposición de arte moderno]]></body>
			<web>https://example.com/evento</web>
			<idrt>VENUE-001</idrt>
			<nombrert><![CDATA[Museo Reina Sofía]]></nombrert>
		</basicData>
		<geoData>
			<address>Calle Santa Isabel 52</address>
			<latitude>40.4085</latitude>
			<longitude>-3.6936</longitude>
		</geoData>
		<multimedia>
			<media>
				<url>https://example.com/image.jpg</url>
			</media>
		</multimedia>
		<extradata>
			<item name="Servicios de pago">Gratuito</item>
			<categorias>
				<categoria>
					<item name="Categoria">Cultura</item>
					<subcategorias>
						<subcategoria>
							<item name="SubCategoria">Exposiciones</item>
						</subcategoria>
					</subcategorias>
				</categoria>
			</categorias>
			<fechas>
				<rango>
					<inicio>20/10/2025</inicio>
					<fin>30/11/2025</fin>
				</rango>
			</fechas>
		</extradata>
	</service>
	<service id="67890" fechaActualizacion="2025-10-19">
		<basicData>
			<name><![CDATA[Concierto de Jazz]]></name>
			<title><![CDATA[Jazz en vivo]]></title>
			<body><![CDATA[Concierto de jazz en directo]]></body>
			<web>https://example.com/concierto</web>
			<idrt>VENUE-002</idrt>
			<nombrert><![CDATA[Café Central]]></nombrert>
		</basicData>
		<geoData>
			<address>Plaza del Ángel 10</address>
			<latitude>40.4153</latitude>
			<longitude>-3.7029</longitude>
		</geoData>
		<multimedia>
			<media>
				<url>https://example.com/jazz.jpg</url>
			</media>
		</multimedia>
		<extradata>
			<item name="Servicios de pago">20€</item>
			<categorias>
				<categoria>
					<item name="Categoria">Música</item>
					<subcategorias>
						<subcategoria>
							<item name="SubCategoria">Jazz</item>
						</subcategoria>
					</subcategorias>
				</categoria>
			</categorias>
			<fechas>
				<rango>
					<inicio>25/10/2025</inicio>
					<fin>25/10/2025</fin>
				</rango>
			</fechas>
		</extradata>
	</service>
</serviceList>`

	// Create mock HTTP server
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(xmlData))
	}))
	defer server.Close()

	// Fetch events
	events, err := FetchEsmadridEvents(server.URL)
	if err != nil {
		t.Fatalf("FetchEsmadridEvents failed: %v", err)
	}

	// Verify User-Agent was set
	expectedUA := "plazaespana-info-site-generator/1.0 (https://github.com/ericphanson/plazaespana.info)"
	if capturedUserAgent != expectedUA {
		t.Errorf("Expected User-Agent %q, got %q", expectedUA, capturedUserAgent)
	}

	// Verify we got 2 events
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	// Verify first event
	event1 := events[0]
	if event1.ID != "12345" {
		t.Errorf("Expected ID '12345', got '%s'", event1.ID)
	}
	if event1.Name != "Exposición de Arte" {
		t.Errorf("Expected Name 'Exposición de Arte', got '%s'", event1.Name)
	}
	if event1.Title != "Arte Contemporáneo" {
		t.Errorf("Expected Title 'Arte Contemporáneo', got '%s'", event1.Title)
	}
	if event1.VenueName != "Museo Reina Sofía" {
		t.Errorf("Expected VenueName 'Museo Reina Sofía', got '%s'", event1.VenueName)
	}
	if event1.Latitude != "40.4085" {
		t.Errorf("Expected Latitude '40.4085', got '%s'", event1.Latitude)
	}
	if event1.Longitude != "-3.6936" {
		t.Errorf("Expected Longitude '-3.6936', got '%s'", event1.Longitude)
	}
	if event1.Category != "Cultura" {
		t.Errorf("Expected Category 'Cultura', got '%s'", event1.Category)
	}
	if event1.Subcategory != "Exposiciones" {
		t.Errorf("Expected Subcategory 'Exposiciones', got '%s'", event1.Subcategory)
	}
	if event1.StartDate != "20/10/2025" {
		t.Errorf("Expected StartDate '20/10/2025', got '%s'", event1.StartDate)
	}
	if event1.EndDate != "30/11/2025" {
		t.Errorf("Expected EndDate '30/11/2025', got '%s'", event1.EndDate)
	}
	if event1.Price != "Gratuito" {
		t.Errorf("Expected Price 'Gratuito', got '%s'", event1.Price)
	}

	// Verify second event
	event2 := events[1]
	if event2.ID != "67890" {
		t.Errorf("Expected ID '67890', got '%s'", event2.ID)
	}
	if event2.Name != "Concierto de Jazz" {
		t.Errorf("Expected Name 'Concierto de Jazz', got '%s'", event2.Name)
	}
	if event2.Category != "Música" {
		t.Errorf("Expected Category 'Música', got '%s'", event2.Category)
	}
	if event2.Subcategory != "Jazz" {
		t.Errorf("Expected Subcategory 'Jazz', got '%s'", event2.Subcategory)
	}
}

// TestFetchEsmadridEvents_HTTPError tests handling of HTTP errors
func TestFetchEsmadridEvents_HTTPError(t *testing.T) {
	// Create server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	// Fetch should return error
	_, err := FetchEsmadridEvents(server.URL)
	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

// TestFetchEsmadridEvents_InvalidXML tests handling of invalid XML
func TestFetchEsmadridEvents_InvalidXML(t *testing.T) {
	// Create server that returns invalid XML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This is not valid XML"))
	}))
	defer server.Close()

	// Fetch should return error
	_, err := FetchEsmadridEvents(server.URL)
	if err == nil {
		t.Fatal("Expected error for invalid XML, got nil")
	}
}

// TestFetchEsmadridEvents_EmptyResponse tests handling of empty service list
func TestFetchEsmadridEvents_EmptyResponse(t *testing.T) {
	// Create server that returns empty service list
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<serviceList>
</serviceList>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(xmlData))
	}))
	defer server.Close()

	// Fetch should succeed with empty list
	events, err := FetchEsmadridEvents(server.URL)
	if err != nil {
		t.Fatalf("Expected success for empty list, got error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

// TestFetchEsmadridEvents_Timeout verifies timeout is set
func TestFetchEsmadridEvents_Timeout(t *testing.T) {
	// Create server that responds normally
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><serviceList></serviceList>`))
	}))
	defer server.Close()

	// This test just verifies the function works with a timeout
	// (testing actual timeout would require waiting 30+ seconds)
	events, err := FetchEsmadridEvents(server.URL)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}
