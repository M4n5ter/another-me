package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/m4n5ter/another-me/internal"
	"github.com/m4n5ter/another-me/internal/mindscape"
	"github.com/m4n5ter/another-me/pkg/config"
)

var configFlag = flag.String("c", "config.toml", "config file path")

func main() {
	flag.Parse()

	parser, err := config.NewParser(*configFlag)
	if err != nil {
		slog.Error("failed to create config parser", "error", err)
		os.Exit(1)
	}

	var config internal.Config
	if err := parser.ParseFile(*configFlag, &config); err != nil {
		slog.Error("failed to parse config", "error", err)
		os.Exit(1)
	}

	mindscapeClient := mindscape.NewClient(config.Mindscape)
	if err := mindscapeClient.Sentinel.HealthCheck(context.Background()); err != nil {
		slog.Error("failed to health check(sentinel)", "error", err)
		os.Exit(1)
	}

	slog.Info("config", "config", config)
}
