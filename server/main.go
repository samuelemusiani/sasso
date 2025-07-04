package main

import (
	"log/slog"
	"os"
	"samuelemusiani/sasso/server/api"
	"samuelemusiani/sasso/server/config"
)

const DEFAULT_LOG_LEVEL = slog.LevelDebug

func main() {
	slog.SetLogLoggerLevel(DEFAULT_LOG_LEVEL)

	// Config file can be passed as the first argument
	if len(os.Args) > 1 {
		err := config.Parse(os.Args[1])
		if err != nil {
			slog.With("error", err).Error("Failed to parse config file")
			os.Exit(1)
		}
	}

	apiLogger := slog.With("module", "api")

	// API
	api.Init(apiLogger)
	api.ListenAndServe(config.Get().Server)
}
