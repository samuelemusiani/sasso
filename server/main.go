package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"samuelemusiani/sasso/pkg/cli"
	"samuelemusiani/sasso/server/api"
	"samuelemusiani/sasso/server/auth"
	"samuelemusiani/sasso/server/config"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/notify"
	"samuelemusiani/sasso/server/proxmox"
)

//go:embed all:_front
var frontFS embed.FS

var (
	// These variables are set at build time using -ldflags "-X main.**=..."
	version = "dev"
	branch  = "develop"
)

func main() {
	clip := cli.NewCli("sasso-server", true, "Path to configuration file (ex. /etc/sasso.yaml)")
	clip.AddCommand("--version", "-v", false, "Print version of binary")
	clip.AddCommand("--change-admin-password", "", true, "Change admin password")

	err := clip.Parse(os.Args)
	if err != nil {
		fmt.Printf("ERROR: %s\n\n%s\n", err.Error(), clip.Help())
		os.Exit(1)
	}

	versionCmd := clip.MustGetCommand("--version")
	if versionCmd.Parsed() {
		fmt.Printf("sasso-server\nVersion: \t%s\nBranch: \t%s\n", version, branch)
		os.Exit(0)
	}

	var (
		haveToChangeAdminPassword bool
		newAdminPassword          string
	)

	adminPasswdCmd := clip.MustGetCommand("--change-admin-password")
	if adminPasswdCmd.Parsed() {
		haveToChangeAdminPassword = true
		newAdminPassword = adminPasswdCmd.Argument()
	}

	var configPath string

	// We parsed the config path
	if !clip.ArgWasParsed() {
		fmt.Printf("ERROR: config path not found\n\n%s", clip.Help())
		os.Exit(1)
	}

	configPath = clip.Argument()

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

	slog.Debug("Parsing config file", "path", configPath)

	err = config.Parse(configPath)
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

	secretKey, err := getSecretKey(c)
	if err != nil {
		slog.Error("Failed to get secrets key", "error", err)
		os.Exit(1)
	}

	realKey, err := base64.StdEncoding.DecodeString(secretKey)
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
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	if haveToChangeAdminPassword {
		err = changeAdminPassword(newAdminPassword)
		if err != nil {
			slog.Error("Failed to change admin password", "error", err)
			os.Exit(1)
		}
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
	}

	// API
	slog.Debug("Initializing API server")

	apiLogger := slog.With("module", "api")

	err = api.Init(apiLogger, realKey, c.Secrets.InternalSecret, frontFS, c.PublicServer, c.PrivateServer, c.PortForwards, c.VPN)
	if err != nil {
		slog.Error("Failed to initialize API server", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	var waitGroup sync.WaitGroup

	waitGroup.Go(func() {
		proxmox.TestEndpointVersion(ctx)
	})
	waitGroup.Go(func() {
		proxmox.TestEndpointClone(ctx)
	})
	waitGroup.Go(func() {
		proxmox.TestEndpointNetZone(ctx)
	})
	waitGroup.Go(func() {
		proxmox.Worker(ctx)
	})
	waitGroup.Go(func() {
		notify.Worker(ctx)
	})

	apiChannelError := api.ListenAndServe()

	select {
	case err = <-apiChannelError:
		slog.Error("Api Server error", "error", err)
		slog.Info("Shutting down due to API server error...")
		cancel()
	case <-ctx.Done():
		slog.Info("Received termination signal, shutting down...")

		pubServerCtx, pubServerCtxCancel := context.WithTimeout(context.Background(), c.PublicServer.ShutdownTimeout)
		privateServerCtx, privateServerCtxCancel := context.WithTimeout(context.Background(), c.PrivateServer.ShutdownTimeout)

		err = api.Shutdown(pubServerCtx, privateServerCtx)
		if err != nil {
			slog.Error("Failed to shut down API server gracefully", "error", err)
		}

		pubServerCtxCancel()
		privateServerCtxCancel()
	}

	waitGroup.Wait()

	if err != nil {
		os.Exit(1)
	}

	slog.Info("Server shut down gracefully")
}

func getSecretKey(c *config.Config) (string, error) {
	if c.Secrets.Key != "" {
		slog.Info("Using secrets key provided in config file")

		return c.Secrets.Key, nil
	} else if c.Secrets.Path != "" {
		slog.Debug("Loading secrets key from file", "path", c.Secrets.Path)

		base64key, err := os.ReadFile(c.Secrets.Path)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to read secrets key file: %w", err)
			}

			slog.Info("Secrets key file does not exist, generating new key", "path", c.Secrets.Path)

			key, err := generateSecretKey(c.Secrets.Path)
			if err != nil {
				return "", fmt.Errorf("failed to generate new secrets key: %w", err)
			}

			return key, nil
		}

		c.Secrets.Key = string(base64key)
	}

	return c.Secrets.Key, nil
}

func generateSecretKey(path string) (string, error) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate new secrets key: %w", err)
	}

	base64key := []byte(base64.StdEncoding.EncodeToString(key))

	slog.Info("Saving key to file", "path", path)

	err = os.WriteFile(path, base64key, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write secrets key to file: %w", err)
	}

	return string(base64key), nil
}

func changeAdminPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password for admin have to be longer than 8 characters")
	}

	err := db.UpdateAdminPassword(password)
	if err != nil {
		return fmt.Errorf("updating admin password on db: %w", err)
	}

	return nil
}
