package main

import (
	"log/slog"
	"os"
	"samuelemusiani/sasso/server/api"
	"samuelemusiani/sasso/server/config"
	"samuelemusiani/sasso/server/db"
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

	c := config.Get()

	// Database
	dbLogger := slog.With("module", "db")
	err := db.Init(dbLogger, c.Database)
	if err != nil {
		slog.With("error", err).Error("Failed to initialize database")
		os.Exit(1)
	}

	// API
	apiLogger := slog.With("module", "api")
	api.Init(apiLogger)
	err = api.ListenAndServe(c.Server)
	if err != nil {
		slog.With("error", err).Error("Failed to start API server")
		os.Exit(1)
	}
}
