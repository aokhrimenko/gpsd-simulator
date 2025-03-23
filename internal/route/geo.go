package route

import (
	"math"
	"time"
)

const earthRadiusMeters = 6371000

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

func radiansToDegrees(rad float64) float64 {
	return rad * 180 / math.Pi
}

func calculateInitialBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)
	deltaLonRad := degreesToRadians(lon2 - lon1)

	x := math.Sin(deltaLonRad) * math.Cos(lat2Rad)
	y := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLonRad)

	initialBearingRad := math.Atan2(x, y)
	initialBearingDeg := radiansToDegrees(initialBearingRad)

	return math.Mod(initialBearingDeg+360, 360)
}

func calculateHaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)
	deltaLat := degreesToRadians(lat2 - lat1)
	deltaLon := degreesToRadians(lon2 - lon1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

// Calculate speed in meters per second given coordinates and time duration
func calculateSpeedMetersPerSecond(lat1, lon1, lat2, lon2 float64, duration time.Duration) float64 {
	distance := calculateHaversineDistance(lat1, lon1, lat2, lon2)
	seconds := duration.Seconds()
	if seconds == 0 {
		return 0
	}
	return distance / seconds
}
