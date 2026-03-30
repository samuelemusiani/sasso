package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"samuelemusiani/sasso/pkg/cli"
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
	clip := cli.NewCli("sasso-router", true, "Path to configuration file (ex. /etc/sasso.yaml)")
	clip.AddCommand("--version", "-v", false, "Print version of binary")

	err := clip.Parse(os.Args)
	if err != nil {
		fmt.Printf("ERROR: %s\n\n%s\n", err.Error(), clip.Help())
		os.Exit(1)
	}

	versionCmd := clip.MustGetCommand("--version")
	if versionCmd.Parsed() {
		fmt.Printf("sasso-router\nVersion: \t%s\nBranch: \t%s\n", version, branch)
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

	var configPath string

	// We parsed the config path
	if !clip.ArgWasParsed() {
		fmt.Printf("ERROR: config path not found\n\n%s", clip.Help())
		os.Exit(1)
	}

	configPath = clip.Argument()

	slog.Debug("Parsing config file", "path", configPath)

	err = config.Parse(configPath)
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
	wg.Go(func() { worker(ctx, workerLogger, c.Server, c.Network, gtw, firewall) })

	<-ctx.Done()
	slog.Info("Shutting down...")

	wg.Wait()
}
