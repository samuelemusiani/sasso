package main

import (
	"fmt"
	"log/slog"
	"os"

	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/util"
	"samuelemusiani/sasso/vpn/wg"
)

const DEFAULT_LOG_LEVEL = slog.LevelDebug

func main() {
	slog.SetLogLoggerLevel(DEFAULT_LOG_LEVEL)

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
			slog.Warn("Invalid LOG_LEVEL value, using default", "value", lLevel, "default", DEFAULT_LOG_LEVEL)
		}
	}

	if len(os.Args) < 2 {
		slog.Error("No config file provided")
		slog.Error("Please provide a config file as the first argument")
		os.Exit(1)
	}

	slog.With("path", os.Args[1]).Debug("Parsing config file")
	err := config.Parse(os.Args[1])
	if err != nil {
		slog.With("error", err).Error("Error parsing config file")
		os.Exit(1)
	}

	c := config.Get()
	slog.With("config", c).Debug("Config file parsed successfully")

	slog.Debug("Initializing Wireguard")
	wireguardLogger := slog.With("module", "wireguard")
	wg.Init(wireguardLogger, &c.Wireguard, c.Wireguard.Interface)

	slog.Debug("Initializing database")
	dbLogger := slog.With("module", "db")
	if err = db.Init(dbLogger, &c.Database); err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	slog.Debug("Initializing utilities")
	utilLogger := slog.With("module", "utils")
	util.Init(utilLogger, c.Wireguard.VPNSubnet)

	workerLogger := slog.With("module", "worker")
	worker(workerLogger, c.Server, c.Firewall)
}
