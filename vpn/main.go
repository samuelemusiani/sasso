package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/fw"
	"samuelemusiani/sasso/vpn/wg"
)

var (
	// These variables are set at build time using -ldflags "-X main.**=..."
	version = "dev"
	branch  = "develop"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" || os.Args[1] == "-v" {
		_, err := fmt.Printf("Sasso VPN\nVersion: \t%s\nBranch: \t%s\n", version, branch)
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
		slog.Error("Error initializing Wireguard", "error", err)
		os.Exit(1)
	}

	slog.Debug("Initializing firewall")

	firewallLogger := slog.With("module", "firewall")

	firewall, err := fw.Init(firewallLogger, c.Firewall)
	if err != nil {
		slog.Error("Error initializing firewall", "error", err)
		os.Exit(1)
	}

	slog.Debug("Initializing database")

	dbLogger := slog.With("module", "db")
	if err = db.Init(dbLogger, &c.Database); err != nil {
		slog.Error("Error initializing database", "error", err)
		os.Exit(1)
	}

	slog.Debug("Initializing utilities")

	if err = checkConfig(c.Server); err != nil {
		slog.Error("Configuration error", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	workerLogger := slog.With("module", "worker")

	var waitGroup sync.WaitGroup
	waitGroup.Go(func() {
		worker(ctx, workerLogger, firewall, c.Server, c.Wireguard.VPNSubnet)
	})

	<-ctx.Done()
	slog.Info("Shutting down...")

	waitGroup.Wait()
}
