package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"os"

	"samuelemusiani/sasso/server/api"
	"samuelemusiani/sasso/server/config"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
)

const DEFAULT_LOG_LEVEL = slog.LevelDebug

func main() {
	slog.SetLogLoggerLevel(DEFAULT_LOG_LEVEL)

	// Config file can be passed as the first argument
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
	// slog.Debug("Config file parsed successfully", "config", c)

	if c.Secrets.Key != "" {
		slog.Info("Using secrets key provided in config file")
	} else if c.Secrets.Path != "" {
		slog.With("path", c.Secrets.Path).Debug("Trying to load secrets key from file")
		base64key, err := os.ReadFile(c.Secrets.Path)
		if err != nil {
			if !os.IsNotExist(err) {
				slog.With("error", err).Error("Failed to read secrets key file")
				os.Exit(1)
			}

			slog.With("path", c.Secrets.Path).Info("Secrets key file does not exist, generating new key")
			_, key, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				slog.With("error", err).Error("Failed to generate new secrets key")
				os.Exit(1)
			}

			base64key = []byte(base64.StdEncoding.EncodeToString(key))

			slog.With("path", c.Secrets.Path).Info("Saving key to file")
			err = os.WriteFile(c.Secrets.Path, base64key, 0600)
			if err != nil {
				slog.With("error", err).Error("Failed to write secrets key to file")
				os.Exit(1)
			}

			c.Secrets.Key = string(base64key)
		}
		c.Secrets.Key = string(base64key)
	} else {
		slog.Error("No secrets key provided in config file or file path")
		slog.Error("Please provide a secrets key in the config file or a path to a file containing the key")
		os.Exit(1)
	}

	real_key, err := base64.StdEncoding.DecodeString(c.Secrets.Key)
	if err != nil {
		slog.With("error", err).Error("Failed to decode secrets key")
		os.Exit(1)
	}

	// Database
	slog.Debug("Initializing database")
	dbLogger := slog.With("module", "db")
	err = db.Init(dbLogger, c.Database)
	if err != nil {
		slog.With("error", err).Error("Failed to initialize database")
		os.Exit(1)
	}

	// Proxmox
	slog.Debug("Initializing proxmox module")
	proxmoxLogger := slog.With("module", "proxmox")
	err = proxmox.Init(proxmoxLogger, c.Proxmox, c.Gateway, c.VPN)
	if err != nil {
		slog.With("error", err).Error("Failed to initialize Proxmox client")
		os.Exit(1)
	}

	slog.Debug("Starting background proxmox tasks")
	go proxmox.TestEndpointVersion()
	go proxmox.TestEndpointClone()
	go proxmox.TestEndpointNetZone()
	go proxmox.TestEndpointGateway()
	go proxmox.TestEndpointVPN()
	go proxmox.Worker()

	// API
	slog.Debug("Initializing API server")
	apiLogger := slog.With("module", "api")
	api.Init(apiLogger, real_key)
	err = api.ListenAndServe(c.Server)
	if err != nil {
		slog.With("error", err).Error("Failed to start API server")
		os.Exit(1)
	}
}
