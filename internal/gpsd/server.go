package gpsd

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"strings"

	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

func NewServer(ctx context.Context, port string, log logger.Logger, routeCtrl *route.Controller) (*Server, error) {
	server := &Server{
		log:       log,
		addr:      ":" + port,
		routeCtrl: routeCtrl,
	}
	server.ctx, server.cancel = context.WithCancel(ctx)
	var err error

	return server, err
}

type Server struct {
	ctx       context.Context
	cancel    context.CancelFunc
	addr      string
	listener  net.Listener
	log       logger.Logger
	routeCtrl *route.Controller
}

func (s *Server) Startup() (err error) {
	s.log.Infof("starting gpsd-simulator server on %s", s.addr)
	s.listener, err = net.Listen("tcp4", s.addr)

	go s.loop()

	return err
}

func (s *Server) Shutdown() {
	s.log.Info("shutting down gpsd-simulator server")
	s.cancel()
}

func (s *Server) loop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	updates, unsubscribeFunc := s.routeCtrl.Subscribe()
	s.log.Infof("Serving %s", conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)
	ctx, cancel := context.WithCancel(s.ctx)
	tpvReportingStarted := false

	defer func() {
		s.log.Infof("Closing connection to %s", conn.RemoteAddr().String())
		unsubscribeFunc()
		cancel()
		conn.Close()
	}()

	_, err := conn.Write([]byte(VersionLine))
	if err != nil {
		s.log.Debug("VersionLine write error:", err)
		return
	}
	s.log.Debugf("Sent: %s", VersionLine)

	for {
		select {
		case <-ctx.Done():
			s.log.Error(ctx.Err())
			return
		default:
		}

		line, err := reader.ReadString(CommandSuffix)
		if err != nil {
			if err != io.EOF {
				s.log.Error("read error:", err)
			}
			s.log.Error("END OF FILE")
			break
		}
		s.log.Debugf("Received: %s", line)
		if strings.HasPrefix(line, WatchCommand) && !tpvReportingStarted {
			tpvReportingStarted = true
			go s.sendTpvReports(ctx, conn, updates)
		}
	}

}

func (s *Server) sendTpvReports(ctx context.Context, conn net.Conn, updates chan route.Point) {
	_, err := conn.Write([]byte(DevicesLine))
	if err != nil {
		s.log.Errorf("DevicesLine write error failed: %v", err)
		return
	}
	s.log.Debugf("Sent: %s", DevicesLine)
	_, err = conn.Write([]byte(WatchLine))
	if err != nil {
		s.log.Errorf("WatchLine write error failed: %v", err)
		return
	}
	s.log.Debugf("Sent: %s", WatchLine)

	for {
		select {
		case <-ctx.Done():
			s.log.Warn(ctx.Err())
			return
		case point, isOpen := <-updates:
			if !isOpen {
				return
			}
			err = WriteTPVReport(conn, point)
			if err != nil {
				s.log.Errorf("sendTpvReports write error failed on point %s: %v", point, err)
				return
			}
		}
	}
}
