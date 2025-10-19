package fetch

// RawEvent represents a single event from Madrid's open data API.
// Field names match the upstream JSON/XML structure exactly.
type RawEvent struct {
	IDEvento          string  `json:"ID-EVENTO" xml:"ID-EVENTO"`
	Titulo            string  `json:"TITULO" xml:"TITULO"`
	Fecha             string  `json:"FECHA" xml:"FECHA"`
	FechaFin          string  `json:"FECHA-FIN" xml:"FECHA-FIN"`
	Hora              string  `json:"HORA" xml:"HORA"`
	NombreInstalacion string  `json:"NOMBRE-INSTALACION" xml:"NOMBRE-INSTALACION"`
	Direccion         string  `json:"DIRECCION" xml:"DIRECCION"`
	Lat               float64 `json:"COORDENADA-LATITUD" xml:"COORDENADA-LATITUD"`
	Lon               float64 `json:"COORDENADA-LONGITUD" xml:"COORDENADA-LONGITUD"`
	ContentURL        string  `json:"CONTENT-URL" xml:"CONTENT-URL"`
	Descripcion       string  `json:"DESCRIPCION" xml:"DESCRIPCION"`
}

// JSONResponse wraps the Madrid API JSON-LD structure.
type JSONResponse struct {
	Context interface{} `json:"@context"`
	Graph   []RawEvent  `json:"@graph"`
}
