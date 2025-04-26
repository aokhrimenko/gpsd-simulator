package http

import (
	"embed"
	_ "embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

//go:embed public/index.html
var indexFile []byte

//go:embed public
var staticFiles embed.FS

type routeRequest struct {
	Name        string  `json:"name"`
	Distance    float64 `json:"distance"`
	Coordinates []struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lng"`
	} `json:"coordinates"`
	MaxSpeed uint `json:"maxSpeed"`
}

func (r *routeRequest) ToPoints() []route.Point {
	points := make([]route.Point, 0, len(r.Coordinates))
	for _, c := range r.Coordinates {
		points = append(points, route.Point{Lat: c.Lat, Lon: c.Lon})
	}
	return points
}

func (s *Server) indexHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write(indexFile); err != nil {
		s.log.Error("HTTP: error writing index.html: ", err)
	}
}

func (s *Server) publicHandler() http.Handler {
	var staticFS = fs.FS(staticFiles)
	htmlContent, err := fs.Sub(staticFS, "public")
	if err != nil {
		log.Fatal(err)
	}
	return http.FileServerFS(htmlContent)
}

type sseMessageInitialRoute struct {
	Type     string        `json:"type"`
	Name     string        `json:"name"`
	Distance float64       `json:"distance"`
	Points   []route.Point `json:"points"`
}

type sseMessageCurrentPoint struct {
	Type   string  `json:"type"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Status string  `json:"status"`
}

func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
	s.log.Infof("HTTP: SSE client connected from %s", r.RemoteAddr)
	// Set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// You may need this locally for CORS requests
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for client disconnection
	clientGone := r.Context().Done()

	updates, cancel := s.routeCtrl.Subscribe()
	defer cancel()

	rc := http.NewResponseController(w)

	if s.routeCtrl.GetRouteSize() > 0 {
		// send the initial route to the client
		err := func() error {
			initialRouteMessage := sseMessageInitialRoute{}
			currentRoute := s.routeCtrl.GetRoute()
			initialRouteMessage.Name = currentRoute.Name
			initialRouteMessage.Distance = currentRoute.Distance
			initialRouteMessage.Points = currentRoute.Points
			initialRouteMessage.Type = "initial-route"
			_, err := w.Write([]byte("data: "))
			if err != nil {
				return err
			}
			err = json.NewEncoder(w).Encode(initialRouteMessage)
			if err != nil {
				return err
			}
			_, err = w.Write([]byte("\n\n"))
			if err != nil {
				return err
			}
			err = rc.Flush()
			return err
		}()
		if err != nil {
			s.log.Error("HTTP: error writing initial route: ", err)
			return
		}
	}

	currentPointMessage := sseMessageCurrentPoint{Type: "current-point"}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-clientGone:
			s.log.Infof("HTTP: SSE client disconnected from %s", r.RemoteAddr)
			return
		case update := <-updates:
			s.routeCtrl.GetState()
			_, err := w.Write([]byte("data: "))
			if err != nil {
				return
			}
			currentPointMessage.Status = s.routeCtrl.GetState().String()
			currentPointMessage.Lat = update.Lat
			currentPointMessage.Lon = update.Lon

			err = json.NewEncoder(w).Encode(currentPointMessage)
			if err != nil {
				return
			}
			_, err = w.Write([]byte("\n\n"))
			if err != nil {
				return
			}
			err = rc.Flush()
			if err != nil {
				return
			}
		}
	}
}

func (s *Server) runHandler(w http.ResponseWriter, _ *http.Request) {
	s.routeCtrl.ToggleState()
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) stopHandler(w http.ResponseWriter, _ *http.Request) {
	s.routeCtrl.UpdateRoute("", 0, 0, []route.Point{})
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) saveRoute(w http.ResponseWriter, r *http.Request) {
	var request routeRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	points := request.ToPoints()
	s.log.Infof("HTTP: Saving route Name=%s, Distance=%f.2, MaxSpeed=%d, Points=%d", request.Name, request.Distance, request.MaxSpeed, len(points))
	s.routeCtrl.UpdateRoute(request.Name, request.Distance, request.MaxSpeed, points)
	w.WriteHeader(http.StatusCreated)
}
