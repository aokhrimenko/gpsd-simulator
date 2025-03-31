package route

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
)

const maxPointsInElevationRequest = 20_000

type openElevationRequestLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type openElevationRequest struct {
	Locations []openElevationRequestLocation `json:"locations"`
}

type openElevationResponse struct {
	Results []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Elevation float64 `json:"elevation"`
	} `json:"results"`
}

func (c *Controller) updateRouteElevations(route *Route) error {
	totalPoints := len(route.Points)
	if totalPoints == 0 {
		return nil
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Process points in batches
	for offset := 0; offset < totalPoints; offset += maxPointsInElevationRequest {
		// Calculate end index for current batch
		end := offset + maxPointsInElevationRequest
		if end > totalPoints {
			end = totalPoints
		}

		// Create request for current batch
		batchSize := end - offset
		request := openElevationRequest{
			Locations: make([]openElevationRequestLocation, batchSize),
		}

		// Fill request with batch points
		for i := 0; i < batchSize; i++ {
			point := route.Points[offset+i]
			request.Locations[i] = openElevationRequestLocation{
				Latitude:  point.Lat,
				Longitude: point.Lon,
			}
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			return err
		}

		c.log.Debugf("Route: making elevation request for %d points (batch %d/%d) of %d bytes",
			len(request.Locations), offset/maxPointsInElevationRequest+1,
			(totalPoints+maxPointsInElevationRequest-1)/maxPointsInElevationRequest, len(jsonData))

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

		if len(response.Results) != batchSize {
			return fmt.Errorf("unexpected number of results: %d, expected %d", len(response.Results), batchSize)
		}

		// Update route points with elevation data
		for _, responseResult := range response.Results {
			for i := offset; i < end; i++ {
				point := route.Points[i]
				if point.Lat == responseResult.Latitude && point.Lon == responseResult.Longitude {
					route.Points[i].Elevation = responseResult.Elevation
					break
				}
			}
		}
	}

	return nil
}
