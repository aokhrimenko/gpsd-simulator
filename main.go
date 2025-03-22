package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aokhrimenko/gpsd-simulator/internal/gpsd"
	"github.com/aokhrimenko/gpsd-simulator/internal/http"
	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

func main() {
	log := logger.NewStdoutLogger()
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()
	signalCtx, signalCancel := signal.NotifyContext(mainCtx, os.Interrupt, syscall.SIGTERM)
	defer signalCancel()

	if len(os.Args) < 3 {
		log.Fatal("Usage: gpsd-simulator <gpsd port> <http port>")
		return
	}

	routeCtrl := route.NewController(mainCtx, time.Second, log)
	defer routeCtrl.Shutdown()

	// start gpsd simulator server
	gpsdServer, err := gpsd.NewServer(mainCtx, os.Args[1], log, routeCtrl)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer gpsdServer.Shutdown()
	if err = gpsdServer.Startup(); err != nil {
		log.Fatal(err)
		return
	}

	// start http server
	httpServer, err := http.NewServer(mainCtx, os.Args[2], log, routeCtrl)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer httpServer.Shutdown()
	go func() {
		if err = httpServer.Startup(); err != nil {
			log.Info(err)
		}
	}()

	<-signalCtx.Done()
	log.Infof("starting graceful shutdown process")
}

//
//func main2() {
//	log := logger.NewStdoutLogger()
//	if len(os.Args) < 3 {
//		log.Fatal("Usage: gpsd-simulator <port> <route-file>")
//	}
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	serverAddress := ":" + os.Args[1]
//
//	routeFile := os.Args[2]
//	r, err := route.ReadRoute(routeFile)
//	if err != nil {
//		log.Fatal(err)
//	}
//	log.Info(r.String())
//
//	l, err := net.Listen("tcp4", serverAddress)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer l.Close()
//
//	ctrl := route.NewController(ctx, time.Second, log)
//
//	usage := func(log logger.Logger) {
//		log.Warnf("To start the controller send %q or %q (without double quotes)", CommandRun, CommandRunShort)
//		log.Warnf("To pause the controller send %q or %q (without double quotes)", CommandPause, CommandPauseShort)
//		log.Warnf("To stop the controller send %q or %q (without double quotes)", CommandStop, CommandStopShort)
//	}
//
//	usage(log)
//
//	go func() {
//		reader := bufio.NewReader(os.Stdin)
//		for {
//			select {
//			case <-ctx.Done():
//				return
//			default:
//			}
//			line, err2 := reader.ReadString('\n')
//			if err2 != nil {
//				log.Error(err2)
//				continue
//			}
//			line = strings.ToLower(strings.TrimSpace(line))
//			switch line {
//			case CommandRun, CommandRunShort:
//				ctrl.Run()
//			case CommandPause, CommandPauseShort:
//				ctrl.Pause()
//			case CommandStop, CommandStopShort:
//				ctrl.Stop()
//			default:
//				log.Warnf("Unknown command: %s", line)
//				usage(log)
//			}
//		}
//	}()
//
//}
