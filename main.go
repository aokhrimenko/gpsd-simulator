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
	"github.com/aokhrimenko/gpsd-simulator/internal/version"
)

var Version = "dev-version"

func main() {
	log := logger.NewStdoutLogger(logger.LevelDebug)
	log.Infof("GPSD Simulator %s", Version)
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()
	signalCtx, signalCancel := signal.NotifyContext(mainCtx, os.Interrupt, syscall.SIGTERM)
	defer signalCancel()

	if len(os.Args) < 3 {
		log.Fatal("Usage: gpsd-simulator <gpsd port> <http port>")
		return
	}

	go version.CheckForUpdate(mainCtx, log, Version)

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
