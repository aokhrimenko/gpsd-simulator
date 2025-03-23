package route

import (
	"context"
	"fmt"
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
	Lat       float64
	Lon       float64
	Speed     float64
	Elevation float64
	Track     float64
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

type Controller struct {
	route      *Route
	listeners  []chan Point
	mu         sync.Mutex
	ctx        context.Context
	cancelFunc context.CancelFunc
	isRunning  bool
	stepDelay  time.Duration
	log        logger.Logger
}

func NewController(parentCtx context.Context, stepDelay time.Duration, log logger.Logger) *Controller {
	c := &Controller{
		route: &Route{
			Points: make([]Point, 0),
			State:  Paused,
		},
		listeners: make([]chan Point, 0),
		stepDelay: stepDelay,
		log:       log,
	}

	c.ctx, c.cancelFunc = context.WithCancel(parentCtx)
	go c.loop()
	return c
}

func (c *Controller) UpdatePoints(points []Point) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.route.Points = make([]Point, len(points))
	for i, point := range points {
		var speed float64
		if i > 0 {
			speed = calculateSpeedMetersPerSecond(points[i-1].Lat, points[i-1].Lon, point.Lat, point.Lon, c.stepDelay)
		}

		var track float64
		if i > 0 {
			track = calculateInitialBearing(points[i-1].Lat, points[i-1].Lon, point.Lat, point.Lon)
		}

		c.route.Points[i] = Point{Lat: point.Lat, Lon: point.Lon, Speed: speed, Track: track}
	}

	if err := c.updateRouteElevations(c.route); err != nil {
		c.log.Error("Route: error updating route elevations: ", err)
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
	c.mu.Lock()
	defer c.mu.Unlock()
	listener := make(chan Point)
	c.listeners = append(c.listeners, listener)

	return listener, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
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

func (c *Controller) broadcast(point Point) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, listener := range c.listeners {
		listener <- point
	}
}

func (c *Controller) loop() {
	stepTimer := time.NewTimer(c.stepDelay)
	for {
		for i := 0; i < len(c.route.Points); i++ {
			select {
			case <-c.ctx.Done():
				c.log.Infof("Route: the controller loop stopped")
				return
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
