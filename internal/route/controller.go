package route

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
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

type LatLon struct {
	Lat, Lon float64
}

type Point struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Speed     float64 `json:"speed"`
	Elevation float64 `json:"elevation"`
	Track     float64 `json:"track"`
}

func (p Point) String() string {
	return fmt.Sprintf("%f,%f", p.Lat, p.Lon)
}

type Route struct {
	Name     string
	Distance float64
	Points   []Point
	State    State
	MaxSpeed uint
}

func (r *Route) String() string {
	return fmt.Sprintf("Route with %d points from %f,%f to %f,%f is currently %s", len(r.Points), r.Points[0].Lat, r.Points[0].Lon, r.Points[len(r.Points)-1].Lat, r.Points[len(r.Points)-1].Lon, r.State)
}

type GeoJsonFile struct {
	Geometry GeoJsonGeometry `json:"geometry"`
}

type GeoJsonGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

func (g GeoJsonFile) Points() []Point {
	points := make([]Point, 0, len(g.Geometry.Coordinates))
	for _, coord := range g.Geometry.Coordinates {
		if len(coord) < 2 {
			continue // Skip invalid coordinates
		}
		points = append(points, Point{
			Lat: coord[1],
			Lon: coord[0],
		})
	}
	return points
}

type Controller struct {
	route         *Route
	listeners     []chan Point
	listenersLock sync.Mutex
	mu            sync.Mutex
	ctx           context.Context
	cancelFunc    context.CancelFunc
	isRunning     bool
	stepDelay     time.Duration
	log           logger.Logger
	stopTheLoop   chan struct{}
}

func NewController(parentCtx context.Context, stepDelay time.Duration, log logger.Logger) *Controller {
	c := &Controller{
		route: &Route{
			Points: make([]Point, 0),
			State:  Paused,
		},
		listeners:   make([]chan Point, 0),
		stepDelay:   stepDelay,
		log:         log,
		stopTheLoop: make(chan struct{}),
	}

	c.ctx, c.cancelFunc = context.WithCancel(parentCtx)
	return c
}

func (c *Controller) Startup() {
	go c.loop()
}

func (c *Controller) CreateRoute(name string, maxSpeed uint, points []Point) Route {
	route := Route{
		Name:     name,
		MaxSpeed: maxSpeed,
		Points:   make([]Point, 0, len(points)),
	}

	maxPointsDistance := float64(0)
	if maxSpeed > 0 {
		maxSpeedMs := float64(maxSpeed) / 3.6
		maxPointsDistance = maxSpeedMs * c.stepDelay.Seconds()
	}

	distances := make(map[LatLon]float64, len(points))

	for i, point := range points {
		var speed float64
		var track float64

		if i > 0 {
			prevIndex := len(route.Points) - 1
			// Skip the same point
			if route.Points[prevIndex].Lat == point.Lat && route.Points[prevIndex].Lon == point.Lon {
				continue
			}
			speed = calculateSpeedMetersPerSecond(route.Points[prevIndex].Lat, route.Points[prevIndex].Lon, point.Lat, point.Lon, c.stepDelay)
			track = calculateInitialBearing(route.Points[prevIndex].Lat, route.Points[prevIndex].Lon, point.Lat, point.Lon)
			pointsDistance := calculateHaversineDistance(route.Points[prevIndex].Lat, route.Points[prevIndex].Lon, point.Lat, point.Lon)
			distances[LatLon{Lat: point.Lat, Lon: point.Lon}] = pointsDistance
			route.Distance += pointsDistance
		}

		route.Points = append(route.Points, Point{Lat: point.Lat, Lon: point.Lon, Speed: speed, Track: track})
	}

	if err := c.updateRouteElevations(&route); err != nil {
		c.log.Error("Route: error updating route elevations: ", err)
	}

	// Tune route points not to reach the maximum speed
	if len(route.Points) > 2 && maxPointsDistance > 0 {
		newPoints := make([]Point, 0, len(route.Points))

		for i, point := range route.Points {
			if i == 0 {
				newPoints = append(newPoints, point)
				continue
			}
			prevIndex := len(newPoints) - 1
			pointsDistance := distances[LatLon{Lat: point.Lat, Lon: point.Lon}]

			if pointsDistance <= maxPointsDistance {
				newPoints = append(newPoints, point)
				continue
			}
			// Calculate number of segments needed based on exact distance
			numSegments := int(math.Ceil(pointsDistance/maxPointsDistance)) - 1

			// Origin point coordinates
			lat1 := newPoints[prevIndex].Lat
			lon1 := newPoints[prevIndex].Lon

			// Calculate bearing once (direction from start to end)
			bearing := calculateInitialBearing(lat1, lon1, point.Lat, point.Lon)
			bearingRad := degreesToRadians(bearing)

			for j := 1; j <= numSegments; j++ {
				// Calculate exact distance from origin for this point
				exactDistance := float64(j) * maxPointsDistance

				// Calculate intermediate point at exactly this distance from origin
				lat1Rad := degreesToRadians(lat1)
				lon1Rad := degreesToRadians(lon1)

				// Angular distance in radians
				angularDistance := exactDistance / earthRadiusMeters

				// Calculate intermediate point using exact distance
				segmentLatRad := math.Asin(math.Sin(lat1Rad)*math.Cos(angularDistance) +
					math.Cos(lat1Rad)*math.Sin(angularDistance)*math.Cos(bearingRad))
				segmentLonRad := lon1Rad + math.Atan2(math.Sin(bearingRad)*math.Sin(angularDistance)*math.Cos(lat1Rad),
					math.Cos(angularDistance)-math.Sin(lat1Rad)*math.Sin(segmentLatRad))

				segmentLat := radiansToDegrees(segmentLatRad)
				segmentLon := radiansToDegrees(segmentLonRad)

				prevIndex = len(newPoints) - 1
				segmentSpeed := calculateSpeedMetersPerSecond(newPoints[prevIndex].Lat, newPoints[prevIndex].Lon, segmentLat, segmentLon, c.stepDelay)
				newPoints = append(newPoints, Point{Lat: segmentLat, Lon: segmentLon, Speed: segmentSpeed, Track: bearing, Elevation: newPoints[prevIndex].Elevation})
			}
		}

		if len(newPoints) > len(route.Points) {
			route.Points = newPoints
		}
	}

	return route
}

func (c *Controller) UpdateRoute(name string, maxSpeed uint, points []Point) {
	newRoute := c.CreateRoute(name, maxSpeed, points)

	c.stopTheLoop <- struct{}{}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.route = &newRoute
	if len(c.route.Points) > 0 {
		c.route.State = Running
	} else {
		c.route.State = Paused
	}

	c.log.Infof("Route: updated route with %d points", len(c.route.Points))
}

func (c *Controller) ToggleState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.route.State == Running {
		c.route.State = Paused
		c.log.Infof("Route: paused")
	} else {
		c.route.State = Running
		c.log.Infof("Route: running")
	}
}

func (c *Controller) Subscribe() (chan Point, func()) {
	c.listenersLock.Lock()
	defer c.listenersLock.Unlock()
	listener := make(chan Point)
	c.listeners = append(c.listeners, listener)

	return listener, func() {
		c.listenersLock.Lock()
		defer c.listenersLock.Unlock()
		for i, l := range c.listeners {
			if l == listener {
				c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
				close(listener)
				return
			}
		}
	}
}

func (c *Controller) GetState() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.route.State
}

func (c *Controller) Shutdown() {
	c.log.Info("Route: shutting down the controller")
	c.cancelFunc()
}

func (c *Controller) GetRouteSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.route.Points)
}

func (c *Controller) GetRoute() Route {
	c.mu.Lock()
	defer c.mu.Unlock()

	clone := Route{
		Name:     c.route.Name,
		Distance: c.route.Distance,
		Points:   make([]Point, len(c.route.Points)),
		State:    c.route.State,
		MaxSpeed: c.route.MaxSpeed,
	}
	copy(clone.Points, c.route.Points)

	return clone
}

func (c *Controller) SetRoute(route Route) {
	c.stopTheLoop <- struct{}{}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.route.Name = route.Name
	c.route.Distance = route.Distance
	c.route.MaxSpeed = route.MaxSpeed
	c.route.Points = make([]Point, len(route.Points))
	copy(c.route.Points, route.Points)

	if len(c.route.Points) > 0 {
		c.route.State = Running
	} else {
		c.route.State = Paused
	}

	c.log.Infof("Route: loaded route with %d points", len(c.route.Points))
}

func (c *Controller) LoadRouteFromFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	var route Route
	if err = json.NewDecoder(file).Decode(&route); err != nil {
		return fmt.Errorf("JSON decode failed: %w", err)
	}
	c.SetRoute(route)

	return nil
}

func (c *Controller) Import(name, inputFile, outputFile string, speed uint) error {
	fmt.Printf("name: %s, input: %s, output: %s, speed: %d\n", name, inputFile, outputFile, speed)
	input, err := os.OpenFile(inputFile, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open input file %s: %w", inputFile, err)
	}
	defer input.Close()

	inputData := GeoJsonFile{}
	if err = json.NewDecoder(input).Decode(&inputData); err != nil {
		return fmt.Errorf("failed to decode input file %s: %w", inputFile, err)
	}

	if name == "" {
		name = fmt.Sprintf("Route %s", time.Now().Format(time.DateTime))
	}

	route := c.CreateRoute(name, speed, inputData.Points())

	if outputFile == "" {
		dir := filepath.Dir(inputFile)
		buf := strings.Builder{}
		buf.WriteString(name)
		buf.WriteString("-")
		if route.Distance > 10000 {
			buf.WriteString(fmt.Sprintf("%.2fkm", route.Distance/1000))
		} else {
			buf.WriteString(fmt.Sprintf("%.0fm", route.Distance))
		}

		if route.MaxSpeed > 0 {
			buf.WriteString(fmt.Sprintf("-%dkmh", route.MaxSpeed))
		}
		buf.WriteString(".json")
		outputFile = filepath.Join(dir, buf.String())
	}

	c.log.Infof("Writing route to %s", outputFile)

	output, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file %s: %w", outputFile, err)
	}
	defer output.Close()

	enc := json.NewEncoder(output)
	enc.SetEscapeHTML(false)
	if err = enc.Encode(route); err != nil {
		return fmt.Errorf("failed to encode route to output file %s: %w", outputFile, err)
	}

	return nil
}

func (c *Controller) broadcast(point Point) {
	c.listenersLock.Lock()
	defer c.listenersLock.Unlock()
	for _, listener := range c.listeners {
		listener <- point
	}
}

func (c *Controller) loop() {
	stepTimer := time.NewTimer(c.stepDelay)
loop:
	for {
		c.mu.Lock()
		pointsLen := len(c.route.Points)
		c.mu.Unlock()

		select {
		case <-c.stopTheLoop:
			continue
		default:
		}

		if pointsLen > 0 {
			c.log.Debugf("Route: starting the loop for %d points", pointsLen)
		} else {
			// No points in the route, wait for new route
			<-stepTimer.C
			stepTimer.Reset(c.stepDelay)
			continue
		}

		for i := 0; i < pointsLen; i++ {
			select {
			case <-c.ctx.Done():
				c.log.Infof("Route: the controller loop stopped")
				return
			case <-c.stopTheLoop:
				c.log.Infof("Route: route need to be updated - stop the loop")
				continue loop
			case <-stepTimer.C:
			}

			var point Point

			switch c.GetState() {
			case Paused:
				if i > 0 {
					i--
				}
				point = c.route.Points[i]
				point.Speed = 0
			case Running:
				point = c.route.Points[i]
			}

			c.broadcast(point)
			stepTimer.Reset(c.stepDelay)
		}
	}
}
