package http

import (
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
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
	Name        string `json:"name"`
	Coordinates []struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lng"`
	} `json:"coordinates"`
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

func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
	s.log.Info("HTTP: SSE client connected")
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
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-clientGone:
			s.log.Info("HTTP: SSE client disconnected")
			return
		case update := <-updates:
			s.routeCtrl.GetState()
			// Send an event to the client
			// Here we send only the "data" field, but there are few others
			_, err := fmt.Fprintf(w, "data: {\"lat\": %f, \"lon\": %f, \"status\": \"%s\"}\n\n", update.Lat, update.Lon, s.routeCtrl.GetState().String())
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
	s.routeCtrl.UpdatePoints([]route.Point{})
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) saveRoute(w http.ResponseWriter, r *http.Request) {
	var request routeRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.routeCtrl.UpdatePoints(request.ToPoints())
	w.WriteHeader(http.StatusCreated)
}
