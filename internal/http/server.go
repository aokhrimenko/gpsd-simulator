package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

type sseMessageType string

const (
	sseMessageTypeInitialRoute sseMessageType = "initial-route"
	sseMessageTypeCurrentPoint sseMessageType = "current-point"
	sseMessageTypeRouteDeleted sseMessageType = "route-deleted"
)

func NewServer(ctx context.Context, port uint, log logger.Logger, routeCtrl *route.Controller) (*Server, error) {
	server := &Server{
		log:            log,
		routeCtrl:      routeCtrl,
		sseBroadcastCh: make([]chan sseMessageType, 0),
	}
	server.ctx, server.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	server.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	mux.HandleFunc("POST /route", server.saveRoute)
	mux.HandleFunc("POST /route/set", server.setRoute)
	mux.HandleFunc("GET /route", server.getRoute)
	mux.HandleFunc("/route/run", server.runHandler)
	mux.HandleFunc("/route/stop", server.stopHandler)
	mux.HandleFunc("/events", server.sseHandler)
	mux.Handle("/", server.publicHandler())

	return server, nil
}

type Server struct {
	ctx            context.Context
	cancel         context.CancelFunc
	log            logger.Logger
	srv            *http.Server
	routeCtrl      *route.Controller
	sseBroadcastCh []chan sseMessageType
	sseBroadcastMu sync.Mutex
}

func (s *Server) Startup() error {
	s.log.Infof("HTTP: starting up server on http://localhost%s/", s.srv.Addr)

	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown() {
	s.log.Info("HTTP: shutting down server")
	s.cancel()
	_ = s.srv.Shutdown(s.ctx)
}

func (s *Server) sseBroadcast(messageType sseMessageType) {
	s.sseBroadcastMu.Lock()
	defer s.sseBroadcastMu.Unlock()

	for _, ch := range s.sseBroadcastCh {
		ch <- messageType
	}
}
