package route

type GeoJson struct {
	Type       string `json:"type"`
	Properties struct {
	} `json:"properties"`
	Geometry struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
	} `json:"geometry"`
}
