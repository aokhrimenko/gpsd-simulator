package route

import (
	"context"
	"fmt"
	"math"
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
	go c.loop()
	return c
}

func (c *Controller) UpdateRoute(name string, distance float64, maxSpeed uint, points []Point) {
	c.stopTheLoop <- struct{}{}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.route.Name = name
	c.route.Distance = distance
	c.route.MaxSpeed = maxSpeed
	maxPointsDistance := float64(0)
	if maxSpeed > 0 {
		maxSpeedMs := float64(maxSpeed) / 3.6
		maxPointsDistance = maxSpeedMs * c.stepDelay.Seconds()
	}

	c.route.Points = make([]Point, 0, len(points))
	for i, point := range points {
		var speed float64
		var track float64

		if i > 0 {
			prevIndex := len(c.route.Points) - 1
			// Skip the same point
			if c.route.Points[prevIndex].Lat == point.Lat && c.route.Points[prevIndex].Lon == point.Lon {
				continue
			}
			speed = calculateSpeedMetersPerSecond(c.route.Points[prevIndex].Lat, c.route.Points[prevIndex].Lon, point.Lat, point.Lon, c.stepDelay)
			track = calculateInitialBearing(c.route.Points[prevIndex].Lat, c.route.Points[prevIndex].Lon, point.Lat, point.Lon)

		}

		c.route.Points = append(c.route.Points, Point{Lat: point.Lat, Lon: point.Lon, Speed: speed, Track: track})
	}

	if err := c.updateRouteElevations(c.route); err != nil {
		c.log.Error("Route: error updating route elevations: ", err)
	}

	// Tune route points not to reach the maximum speed
	if len(c.route.Points) > 2 && maxPointsDistance > 0 {
		newPoints := make([]Point, 0, len(c.route.Points))

		for i, point := range c.route.Points {
			if i == 0 {
				newPoints = append(newPoints, point)
				continue
			}
			prevIndex := len(newPoints) - 1
			pointsDistance := calculateHaversineDistance(newPoints[prevIndex].Lat, newPoints[prevIndex].Lon, point.Lat, point.Lon)

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

		if len(newPoints) > len(c.route.Points) {
			c.route.Points = newPoints
		}
	}

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
