package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/fw"
	"samuelemusiani/sasso/router/gateway"
)

var (
	// These variables are set at build time using -ldflags "-X main.**=..."
	version = "dev"
	branch  = "develop"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" || os.Args[1] == "-v" {
		_, err := fmt.Printf("Sasso Router\nVersion: \t%s\nBranch: \t%s\n", version, branch)
		if err != nil {
			os.Exit(1)
		}

		os.Exit(0)
	}

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

	gatewayLogger := slog.With("module", "gateway")

	err = gateway.Init(gatewayLogger, c.Gateway)
	if err != nil {
		slog.Error("Failed to initialize gateway", "error", err)
		os.Exit(1)
	}

	gtw := gateway.Get()
	if gtw == nil {
		slog.Error("Gateway is not initialized")
		os.Exit(1)
	}

	fwLogger := slog.With("module", "firewall")

	err = fw.Init(fwLogger, c.Firewall)
	if err != nil {
		slog.Error("Failed to initialize firewall", "error", err)
		os.Exit(1)
	}

	firewall := fw.Get()
	if firewall == nil {
		slog.Error("Firewall is not initialized")
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	workerLogger := slog.With("module", "worker")

	var wg sync.WaitGroup
	wg.Go(func() { worker(ctx, workerLogger, c.Server, gtw, firewall) })

	<-ctx.Done()
	slog.Info("Shutting down...")

	wg.Wait()
}
