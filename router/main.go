package main

import (
	"log/slog"
	"os"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/fw"
	"samuelemusiani/sasso/router/gateway"
	"samuelemusiani/sasso/router/utils"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	lLevel, ok := os.LookupEnv("LOG_LEVEL")
	if ok {
		switch lLevel {
		case "DEBUG":
			slog.SetLogLoggerLevel(slog.LevelDebug)
		case "INFO":
			slog.SetLogLoggerLevel(slog.LevelInfo)
		case "WARN":
			slog.SetLogLoggerLevel(slog.LevelWarn)
		case "ERROR":
			slog.SetLogLoggerLevel(slog.LevelError)
		default:
			slog.Warn("Invalid LOG_LEVEL value, using default debug", "value", lLevel)
		}
	}

	// Config file must be passed as the first argument
	if len(os.Args) <= 1 {
		slog.Error("No config file provided")
		slog.Error("Please provide a config file as the first argument")
		os.Exit(1)
	}

	slog.Debug("Parsing config file", "path", os.Args[1])
	err := config.Parse(os.Args[1])
	if err != nil {
		slog.Error("Failed to parse config file", "error", err)
		os.Exit(1)
	}

	c := config.Get()
	slog.Debug("Config file parsed successfully", "config", c)

	slog.Debug("Initializing utilities")
	utilsLogger := slog.With("module", "utils")
	err = utils.Init(utilsLogger, c.Network)
	if err != nil {
		slog.Error("Failed to initialize utilities", "error", err)
		os.Exit(1)
	}

	gatewayLogger := slog.With("module", "gateway")
	err = gateway.Init(gatewayLogger, c.Gateway)
	if err != nil {
		slog.Error("Failed to initialize gateway", "error", err)
		os.Exit(1)
	}

	fwLogger := slog.With("module", "firewall")
	err = fw.Init(fwLogger, c.Firewall)
	if err != nil {
		slog.Error("Failed to initialize firewall", "error", err)
		os.Exit(1)
	}

	// Database
	slog.Debug("Initializing database")
	dbLogger := slog.With("module", "db")
	err = db.Init(dbLogger, c.Database)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	if err = checkConfig(c.Server); err != nil {
		slog.Error("Configuration error", "error", err)
		os.Exit(1)
	}

	workerLogger := slog.With("module", "worker")
	worker(workerLogger, c.Server)
}
