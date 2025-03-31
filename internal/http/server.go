package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

func NewServer(ctx context.Context, port string, log logger.Logger, routeCtrl *route.Controller) (*Server, error) {
	server := &Server{
		log:       log,
		port:      port,
		routeCtrl: routeCtrl,
	}
	server.ctx, server.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	server.srv = &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	mux.HandleFunc("POST /route", server.saveRoute)
	mux.HandleFunc("/route/run", server.runHandler)
	mux.HandleFunc("/route/stop", server.stopHandler)
	mux.HandleFunc("/events", server.sseHandler)
	mux.Handle("/", server.publicHandler())

	return server, nil
}

type Server struct {
	ctx       context.Context
	cancel    context.CancelFunc
	log       logger.Logger
	port      string
	srv       *http.Server
	routeCtrl *route.Controller
}

func (s *Server) Startup() error {
	s.log.Infof("HTTP: starting up server on %s", s.srv.Addr)

	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown() {
	s.log.Info("HTTP: shutting down server")
	s.cancel()
	_ = s.srv.Shutdown(s.ctx)
}
