package route

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
)

type openElevationRequestLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type openElevationRequest struct {
	Locations []openElevationRequestLocation `json:"locations"`
}

type openElevationResponse struct {
	Results []struct {
		Longitude float64 `json:"longitude"`
		Elevation float64 `json:"elevation"`
		Latitude  float64 `json:"latitude"`
	} `json:"results"`
}

func (c *Controller) updateRouteElevations(route *Route) error {
	request := openElevationRequest{Locations: make([]openElevationRequestLocation, len(route.Points))}
	for i, point := range route.Points {
		request.Locations[i] = openElevationRequestLocation{Latitude: point.Lat, Longitude: point.Lon}
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Post("https://api.open-elevation.com/api/v1/lookup", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get elevation data: %s", resp.Status)
	}

	var response openElevationResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if len(response.Results) != len(route.Points) {
		return fmt.Errorf("unexpected number of results: %d", len(response.Results))
	}

	for _, responseResult := range response.Results {
		for i, point := range route.Points {
			if point.Lat == responseResult.Latitude && point.Lon == responseResult.Longitude {
				route.Points[i].Elevation = responseResult.Elevation
				break
			}
		}
	}

	return nil
}
