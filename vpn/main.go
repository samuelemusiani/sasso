package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/util"
	"samuelemusiani/sasso/vpn/wg"
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

	if len(os.Args) < 2 {
		slog.Error("No config file provided")
		slog.Error("Please provide a config file as the first argument")
		os.Exit(1)
	}

	slog.Debug("Parsing config file", "path", os.Args[1])

	err := config.Parse(os.Args[1])
	if err != nil {
		slog.Error("Error parsing config file", "error", err)
		os.Exit(1)
	}

	c := config.Get()
	slog.Debug("Config file parsed successfully", "config", c)

	slog.Debug("Initializing Wireguard")

	wireguardLogger := slog.With("module", "wireguard")

	err = wg.Init(wireguardLogger, &c.Wireguard)
	if err != nil {
		fmt.Printf("Error initializing Wireguard: %v\n", err)
		os.Exit(1)
	}

	slog.Debug("Initializing database")

	dbLogger := slog.With("module", "db")
	if err = db.Init(dbLogger, &c.Database); err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	slog.Debug("Initializing utilities")

	utilLogger := slog.With("module", "utils")
	util.Init(utilLogger)

	if err = checkConfig(c.Server, c.Firewall, c.Wireguard.VPNSubnet); err != nil {
		slog.Error("Configuration error", "error", err)
		os.Exit(1)
	}

	if err = checkFirewallStatus(c.Firewall); err != nil {
		slog.Error("Firewall configuration error", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	workerLogger := slog.With("module", "worker")

	var waitGroup sync.WaitGroup
	waitGroup.Go(func() {
		worker(ctx, workerLogger, c.Server, c.Firewall, c.Wireguard.VPNSubnet)
	})

	<-ctx.Done()
	slog.Info("Shutting down...")

	waitGroup.Wait()
}
