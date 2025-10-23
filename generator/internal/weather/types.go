package weather

import "time"

// Weather represents weather information for a specific event date
type Weather struct {
	Date            string  `json:"date"`             // Forecast date (YYYY-MM-DD)
	TempMax         int     `json:"temp_max"`         // Max temp (°C)
	TempMin         int     `json:"temp_min"`         // Min temp (°C)
	PrecipProb      int     `json:"precip_prob"`      // Precipitation probability (%)
	PrecipAmount    float64 `json:"precip_amount"`    // Total precipitation (mm)
	SkyCode         string  `json:"sky_code"`         // AEMET sky state code (e.g., "12", "15n")
	SkyDescription  string  `json:"sky_description"`  // Human-readable sky state (Spanish)
	SkyIconURL      string  `json:"sky_icon_url"`     // Weather icon URL
	WeatherCategory string  `json:"weather_category"` // Simplified category for CSS (clear/cloudy/rain/etc)
	IsNight         bool    `json:"is_night"`         // True if code ends with 'n'
}

// Forecast represents the full AEMET forecast response
type Forecast struct {
	Origin     Origin     `json:"origen"`
	Elaborated time.Time  `json:"elaborado"`
	Name       string     `json:"nombre"`
	Province   string     `json:"provincia"`
	Prediction Prediction `json:"prediccion"`
}

// Origin contains AEMET metadata
type Origin struct {
	Producer  string `json:"productor"`
	Web       string `json:"web"`
	Link      string `json:"enlace"`
	Language  string `json:"language"`
	Copyright string `json:"copyright"`
	LegalNote string `json:"notaLegal"`
}

// Prediction contains the forecast days
type Prediction struct {
	Days []DayForecast `json:"dia"`
}

// DayForecast represents a single day's forecast
type DayForecast struct {
	Date              string             `json:"fecha"`
	Sunrise           string             `json:"orto"`
	Sunset            string             `json:"ocaso"`
	Temperature       Temperature        `json:"temperatura"`
	SkyState          []PeriodValue      `json:"estadoCielo"`
	Precipitation     []PeriodFloatValue `json:"precipitacion"`
	PrecipProbability []PeriodIntValue   `json:"probPrecipitacion"`
	Wind              []Wind             `json:"viento"`
	MaxGust           []PeriodIntValue   `json:"rachaMax"`
	RelativeHumidity  Humidity           `json:"humedadRelativa"`
	ThermalSensation  ThermalSensation   `json:"sensTermica"`
	UVMax             int                `json:"uvMax"`
	SnowLevel         []PeriodValue      `json:"cotaNieveProv"`
}

// Temperature contains temperature data for a day
type Temperature struct {
	Max  int                 `json:"maxima"`
	Min  int                 `json:"minima"`
	Data []HourlyTemperature `json:"dato"`
}

// HourlyTemperature represents temperature at a specific hour
type HourlyTemperature struct {
	Value int `json:"value"`
	Hour  int `json:"hora"`
}

// PeriodValue represents a string value for a time period
type PeriodValue struct {
	Value       string `json:"value"`
	Period      string `json:"periodo"`
	Description string `json:"descripcion"`
}

// PeriodIntValue represents an integer value for a time period
type PeriodIntValue struct {
	Value  int    `json:"value"`
	Period string `json:"periodo"`
}

// PeriodFloatValue represents a float value for a time period
type PeriodFloatValue struct {
	Value  float64 `json:"value"`
	Period string  `json:"periodo"`
}

// Wind represents wind data for a period
type Wind struct {
	Direction string `json:"direccion"`
	Speed     int    `json:"velocidad"`
	Period    string `json:"periodo"`
}

// Humidity contains humidity data
type Humidity struct {
	Max  int              `json:"maxima"`
	Min  int              `json:"minima"`
	Data []HourlyHumidity `json:"dato"`
}

// HourlyHumidity represents humidity at a specific hour
type HourlyHumidity struct {
	Value int `json:"value"`
	Hour  int `json:"hora"`
}

// ThermalSensation contains thermal sensation data
type ThermalSensation struct {
	Max  int                      `json:"maxima"`
	Min  int                      `json:"minima"`
	Data []HourlyThermalSensation `json:"dato"`
}

// HourlyThermalSensation represents thermal sensation at a specific hour
type HourlyThermalSensation struct {
	Value int `json:"value"`
	Hour  int `json:"hora"`
}

// MetadataResponse represents the first-step AEMET API response
type MetadataResponse struct {
	Description string `json:"descripcion"`
	State       int    `json:"estado"`
	DataURL     string `json:"datos"`
	MetadataURL string `json:"metadatos"`
}
