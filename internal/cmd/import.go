package cmd

import (
	"context"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/aokhrimenko/gpsd-simulator/internal/gpsd"
	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

type importConfig struct {
	Debug      bool
	Verbose    bool
	Name       string
	InputFile  string
	OutputFile string
	Speed      uint
}

func Import(currentVersion string) *cobra.Command {
	importCfg := &importConfig{}
	writerCfg := gpsd.WriterConfig{}
	var rootCmd = &cobra.Command{
		Use:     "import",
		Version: currentVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeImportCommand(currentVersion, importCfg, writerCfg)
		},
	}
	rootCmd.Flags().StringVarP(&importCfg.Name, "name", "n", "", "Route name")
	rootCmd.Flags().StringVarP(&importCfg.InputFile, "input", "i", "", "Path to the input GeoJSON file")
	rootCmd.Flags().StringVarP(&importCfg.OutputFile, "output", "o", "", "Path to the output gpsd route file")
	rootCmd.Flags().UintVarP(&importCfg.Speed, "speed", "s", 0, "Speed in km/h for the route (default is 0, which means no speed limit)")
	rootCmd.Flags().BoolVarP(&importCfg.Debug, "debug", "d", false, "Enable debug logging")
	rootCmd.Flags().BoolVarP(&importCfg.Verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.Flags().SortFlags = false
	_ = rootCmd.MarkFlagRequired("input")
	return rootCmd
}

func executeImportCommand(currentVersionString string, cfg *importConfig, writerCfg gpsd.WriterConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx

	logLevel := logger.LevelInfo
	if cfg.Verbose {
		logLevel = logger.LevelVerbose
	} else if cfg.Debug {
		logLevel = logger.LevelDebug
	}

	log := logger.NewStdoutLogger(logLevel)
	currentVersion, err := semver.NewVersion(currentVersionString)
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Infof("GPSD Simulator v%s", currentVersion.String())

	routeCtrl := route.NewController(ctx, time.Second, log)
	defer routeCtrl.Shutdown()

	err = routeCtrl.Import(cfg.Name, cfg.InputFile, cfg.OutputFile, cfg.Speed)
	if err != nil {
		log.Error("Failed to import route:", err)
	}

	return nil
}
