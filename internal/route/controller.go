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

func (c *Controller) UpdatePoints(points []Point) {
	c.stopTheLoop <- struct{}{}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.route.Points = make([]Point, len(points))
	for i, point := range points {
		var speed float64
		var track float64

		if i > 0 {
			if points[i-1].Lat == point.Lat && points[i-1].Lon == point.Lon {
				speed = points[i-1].Speed
				track = points[i-1].Track
			} else {
				speed = calculateSpeedMetersPerSecond(points[i-1].Lat, points[i-1].Lon, point.Lat, point.Lon, c.stepDelay)
				track = calculateInitialBearing(points[i-1].Lat, points[i-1].Lon, point.Lat, point.Lon)
			}
		}

		c.route.Points[i] = Point{Lat: point.Lat, Lon: point.Lon, Speed: speed, Track: track}
	}

	if err := c.updateRouteElevations(c.route); err != nil {
		c.log.Error("Route: error updating route elevations: ", err)
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

		if pointsLen > 0 {
			c.log.Debugf("Route: starting the loop for %d points", pointsLen)
		}

		select {
		case <-c.stopTheLoop:
			continue
		default:
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
