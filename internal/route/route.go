package route

import (
	"fmt"
)

type State uint8

func (s State) String() string {
	switch s {
	case Paused:
		return "Paused"
	case Running:
		return "Running"
	default:
		return "Unknown"
	}
}

const (
	Paused State = iota
	Running
)

type Point struct {
	Lat       float64
	Lon       float64
	Speed     float64
	Elevation float64
}

func (p Point) String() string {
	return fmt.Sprintf("%f,%f", p.Lat, p.Lon)
}

type Route struct {
	Points []Point
	State  State
}

func (r *Route) String() string {
	return fmt.Sprintf("Route with %d points from %f,%f to %f,%f is currently %s", len(r.Points), r.Points[0].Lat, r.Points[0].Lon, r.Points[len(r.Points)-1].Lat, r.Points[len(r.Points)-1].Lon, r.State)
}
