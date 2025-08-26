package main

import (
	"log/slog"
	"os"
	"samuelemusiani/sasso/router/api"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/gateway"
	"samuelemusiani/sasso/router/ticket"
	"samuelemusiani/sasso/router/utils"
)

const DEFAULT_LOG_LEVEL = slog.LevelDebug

func main() {
	slog.SetLogLoggerLevel(DEFAULT_LOG_LEVEL)

	// Config file must be passed as the first argument
	if len(os.Args) <= 1 {
		slog.Error("No config file provided")
		slog.Error("Please provide a config file as the first argument")
		os.Exit(1)
	}

	slog.Debug("Parsing config file", "path", os.Args[1])
	err := config.Parse(os.Args[1])
	if err != nil {
		slog.With("error", err).Error("Failed to parse config file")
		os.Exit(1)
	}

	c := config.Get()
	slog.Debug("Config file parsed successfully", "config", c)

	slog.Debug("Initializing utilities")
	utilsLogger := slog.With("module", "utils")
	err = utils.Init(utilsLogger, c.Network)
	if err != nil {
		slog.With("error", err).Error("Failed to initialize utilities")
		os.Exit(1)
	}

	// Ticketing
	ticketLogger := slog.With("module", "ticket")
	ticket.Init(ticketLogger)

	gatewayLogger := slog.With("module", "gateway")
	err = gateway.Init(gatewayLogger, c.Gateway)

	// Database
	slog.Debug("Initializing database")
	dbLogger := slog.With("module", "db")
	err = db.Init(dbLogger, c.Database)
	if err != nil {
		slog.With("error", err).Error("Failed to initialize database")
		os.Exit(1)
	}

	// API
	slog.Debug("Initializing API server")
	apiLogger := slog.With("module", "api")
	err = api.Init(apiLogger, c.Api.Secret)
	if err != nil {
		slog.With("error", err).Error("Failed to initialize API server")
		os.Exit(1)
	}
	err = api.ListenAndServe(c.Server)
	if err != nil {
		slog.With("error", err).Error("Failed to start API server")
		os.Exit(1)
	}
}
