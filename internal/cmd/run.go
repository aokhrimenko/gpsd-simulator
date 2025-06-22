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

type mainConfig struct {
	GpsdPort  uint
	WebUiPort uint
	Debug     bool
	Verbose   bool
	File      string
}

func Run(currentVersion string) *cobra.Command {
	mainCfg := &mainConfig{}
	writerCfg := gpsd.WriterConfig{}
	var runCmd = &cobra.Command{
		Use:     "run",
		Version: currentVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRunCommand(currentVersion, mainCfg, writerCfg)
		},
	}
	runCmd.Flags().UintVarP(&mainCfg.GpsdPort, "gpsd-port", "g", 2947, "Port for the GPSD server")
	runCmd.Flags().UintVarP(&mainCfg.WebUiPort, "webui-port", "w", 8881, "Port for the web UI")
	runCmd.Flags().BoolVarP(&mainCfg.Debug, "debug", "d", false, "Enable debug logging")
	runCmd.Flags().BoolVarP(&mainCfg.Verbose, "verbose", "v", false, "Enable verbose logging")
	runCmd.Flags().StringVarP(&mainCfg.File, "file", "f", "", "Path to the route file (JSON format)")

	// WriterConfig
	runCmd.Flags().StringVar(&writerCfg.VersionRelease, "version-release", gpsd.DefaultVersionRelease, "VERSION/release field")
	runCmd.Flags().StringVar(&writerCfg.VersionRev, "version-revision", gpsd.DefaultVersionRev, "VERSION/rev field")
	runCmd.Flags().UintVar(&writerCfg.VersionProtoMajor, "version-proto-major", gpsd.DefaultVersionProtoMajor, "VERSION/proto_major field")
	runCmd.Flags().UintVar(&writerCfg.VersionProtoMinor, "version-proto-minor", gpsd.DefaultVersionProtoMinor, "VERSION/proto_minor field")
	runCmd.Flags().StringVar(&writerCfg.DevicePath, "device-path", gpsd.DefaultVersionDevicePath, "DEVICES/devices/path field")
	runCmd.Flags().StringVar(&writerCfg.DeviceDriver, "device-driver", gpsd.DefaultDeviceDriver, "DEVICES/devices/driver field")
	runCmd.Flags().StringVar(&writerCfg.DeviceActivated, "device-activated", gpsd.DefaultDeviceActivated, "DEVICES/devices/activated field")
	runCmd.Flags().UintVar(&writerCfg.DeviceBps, "device-bps", gpsd.DefaultDeviceBps, "DEVICES/devices/bps field")
	runCmd.Flags().StringVar(&writerCfg.DeviceParity, "device-parity", gpsd.DefaultDeviceParity, "DEVICES/devices/parity field")
	runCmd.Flags().UintVar(&writerCfg.DeviceStopBits, "device-stop-bits", gpsd.DefaultDeviceStopBits, "DEVICES/devices/stopbits field")
	runCmd.Flags().UintVar(&writerCfg.TpvMode, "tpv-mode", gpsd.DefaultTpvMode, "TPV/mode field")

	runCmd.Flags().SortFlags = false
	return runCmd
}

func executeRunCommand(currentVersionString string, mainCfg *mainConfig, writerCfg gpsd.WriterConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx

	logLevel := logger.LevelInfo
	if mainCfg.Verbose {
		logLevel = logger.LevelVerbose
	} else if mainCfg.Debug {
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
	routeCtrl.Startup()
	defer routeCtrl.Shutdown()

	// start gpsd simulator server
	gpsdServer, err := gpsd.NewServer(ctx, mainCfg.GpsdPort, log, routeCtrl, writerCfg)
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
	httpServer, err := http.NewServer(ctx, mainCfg.WebUiPort, log, routeCtrl)
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

	// try to load route from file if specified
	if err = routeCtrl.LoadRouteFromFile(mainCfg.File); err != nil {
		log.Errorf("error loading route from file %s: %v", mainCfg.File, err)
	}

	<-signalCtx.Done()
	log.Infof("starting graceful shutdown process")
	return nil
}
