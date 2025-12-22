package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"samuelemusiani/sasso/server/api"
	"samuelemusiani/sasso/server/auth"
	"samuelemusiani/sasso/server/config"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/notify"
	"samuelemusiani/sasso/server/proxmox"
)

//go:embed all:_front
var frontFS embed.FS

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

	// Config file can be passed as the first argument
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
	// slog.Debug("Config file parsed successfully", "config", c)

	if c.Secrets.Key == "" && c.Secrets.Path == "" {
		slog.Error("No secrets key provided in config file or file path")
		slog.Error("Please provide a secrets key in the config file or a path to a file containing the key")
		os.Exit(1)
	}

	realKey, err := base64.StdEncoding.DecodeString(getSecretKey(c))
	if err != nil {
		slog.Error("Failed to decode secrets key", "error", err)
		os.Exit(1)
	}

	frontFS, err := fs.Sub(frontFS, "_front")
	if err != nil {
		slog.Error("Initializing change base path for front fs", "err", err)
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

	// Auth
	authLogger := slog.With("module", "auth")

	err = auth.Init(authLogger)
	if err != nil {
		slog.Error("Failed to initialize authentication module", "error", err)
		os.Exit(1)
	}

	// Proxmox init
	slog.Debug("Initializing proxmox module")

	proxmoxLogger := slog.With("module", "proxmox")

	err = proxmox.Init(proxmoxLogger, c.Proxmox)
	if err != nil {
		slog.Error("Failed to initialize Proxmox client", "error", err)
		os.Exit(1)
	}

	// Notifications
	if c.Notifications.Enabled {
		notifyLogger := slog.With("module", "notify")

		err = notify.Init(notifyLogger, c.Notifications)
		if err != nil {
			slog.Error("Failed to initialize notifications module", "error", err)
			os.Exit(1)
		}

		notify.StartWorker()
	}

	// API
	slog.Debug("Initializing API server")

	apiLogger := slog.With("module", "api")

	err = api.Init(apiLogger, realKey, c.Secrets.InternalSecret, frontFS, c.PublicServer, c.PrivateServer, c.PortForwards, c.VPN)
	if err != nil {
		slog.Error("Failed to initialize API server", "error", err)
		os.Exit(1)
	}

	// Proxmox workers start
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	slog.Debug("Starting background proxmox tasks")

	go proxmox.TestEndpointVersion()
	go proxmox.TestEndpointClone()
	go proxmox.TestEndpointNetZone()

	proxmox.StartWorker()

	channelError := make(chan error, 1)

	go func() {
		err = api.ListenAndServe()
		if err != nil {
			slog.Error("Failed to start API server", "error", err)
		}

		channelError <- err
	}()

	select {
	case err := <-channelError:
		slog.Error("Server error", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		slog.Info("Received termination signal, shutting down...")

		var waitGroup sync.WaitGroup
		waitGroup.Add(3)

		go func() {
			defer waitGroup.Done()

			err := api.Shutdown()
			if err != nil {
				slog.Error("Failed to shut down API server", "error", err)
			}
		}()

		go func() {
			defer waitGroup.Done()

			err = proxmox.ShutdownWorker()
			if err != nil {
				slog.Error("Failed to shut down Proxmox worker", "error", err)
			}
		}()

		go func() {
			defer waitGroup.Done()

			err = notify.ShutdownWorker()
			if err != nil {
				slog.Error("Failed to shut down notifications worker", "error", err)
			}
		}()

		waitGroup.Wait()
	}

	slog.Info("Server shut down gracefully")
	os.Exit(0)
}

func getSecretKey(c *config.Config) string {
	if c.Secrets.Key != "" {
		slog.Info("Using secrets key provided in config file")

		return c.Secrets.Key
	} else if c.Secrets.Path != "" {
		slog.Debug("Loading secrets key from file", "path", c.Secrets.Path)

		base64key, err := os.ReadFile(c.Secrets.Path)
		if err != nil {
			if !os.IsNotExist(err) {
				slog.Error("Failed to read secrets key file", "error", err)
				os.Exit(1)
			}

			slog.Info("Secrets key file does not exist, generating new key", "path", c.Secrets.Path)

			return generateSecretKey(c.Secrets.Path)
		}

		c.Secrets.Key = string(base64key)
	}

	return c.Secrets.Key
}

func generateSecretKey(path string) string {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		slog.Error("Failed to generate new secrets key", "error", err)
		os.Exit(1)
	}

	base64key := []byte(base64.StdEncoding.EncodeToString(key))

	slog.Info("Saving key to file", "path", path)

	err = os.WriteFile(path, base64key, 0600)
	if err != nil {
		slog.Error("Failed to write secrets key to file", "error", err)
		os.Exit(1)
	}

	return string(base64key)
}
