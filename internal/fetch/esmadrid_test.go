package fetch

import (
	"encoding/xml"
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
	data, err := os.ReadFile("../../test/fixtures/esmadrid-agenda.xml")
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
