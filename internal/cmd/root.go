package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/aokhrimenko/gpsd-simulator/internal/gpsd"
	"github.com/aokhrimenko/gpsd-simulator/internal/http"
	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
	"github.com/aokhrimenko/gpsd-simulator/internal/route"
	"github.com/aokhrimenko/gpsd-simulator/internal/version"
)

type cmdFlags struct {
	GpsdPort  uint
	WebUiPort uint
	RouteFile string
	Debug     bool
	Verbose   bool
}

func Root(currentVersion string) *cobra.Command {

	flags := &cmdFlags{}
	var rootCmd = &cobra.Command{
		Use:   "gpsd-simulator",
		Short: "GPSD simulator",
		//Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApp(currentVersion, flags)
		},
	}
	rootCmd.Flags().UintVarP(&flags.GpsdPort, "gpsd-port", "g", 2947, "Port for the GPSD server")
	rootCmd.Flags().UintVarP(&flags.WebUiPort, "webui-port", "w", 8881, "Port for the web UI")
	rootCmd.Flags().StringVarP(&flags.RouteFile, "route-file", "r", "", "Path to the route file to import")
	rootCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "Enable debug logging")
	rootCmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().SortFlags = false
	return rootCmd
}

func runApp(currentVersionString string, flags *cmdFlags) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx

	logLevel := logger.LevelInfo
	if flags.Verbose {
		logLevel = logger.LevelVerbose
	} else if flags.Debug {
		logLevel = logger.LevelDebug
	}

	log := logger.NewStdoutLogger(logLevel)
	currentVersion, err := semver.NewVersion(currentVersionString)
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Infof("GPSD Simulator v%s", currentVersion.String())

	signalCtx, signalCancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer signalCancel()

	go version.CheckForUpdate(ctx, log, currentVersion)

	routeCtrl := route.NewController(ctx, time.Second, log)
	defer routeCtrl.Shutdown()

	// start gpsd simulator server
	gpsdServer, err := gpsd.NewServer(ctx, flags.GpsdPort, log, routeCtrl)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer gpsdServer.Shutdown()
	if err = gpsdServer.Startup(); err != nil {
		log.Fatal(err)
		return err
	}

	// start http server
	httpServer, err := http.NewServer(ctx, flags.WebUiPort, log, routeCtrl)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer httpServer.Shutdown()
	go func() {
		if err = httpServer.Startup(); err != nil {
			log.Info(err)
		}
	}()

	<-signalCtx.Done()
	log.Infof("starting graceful shutdown process")
	return nil
}
