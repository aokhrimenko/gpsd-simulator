package route

import (
	"context"
	"sync"
	"time"

	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
)

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
		c.route.Points[i] = Point{Lat: point.Lat, Lon: point.Lon, Speed: 15}
	}

	if err := c.updateRouteElevations(c.route); err != nil {
		c.log.Error("error updating route elevations: ", err)
	}
}

func (c *Controller) ToggleState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.route.State == Running {
		c.route.State = Paused
	} else {
		c.route.State = Running
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
	c.log.Info("shutting down route controller")
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
				c.log.Infof("route controller loop stopped")
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
