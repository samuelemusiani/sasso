package main

import (
	// "context"
	// "crypto/ed25519"
	// "crypto/rand"
	"embed"
	// "encoding/base64"
	// "io/fs"
	"log/slog"
	"os"
	// "os/signal"
	// "sync"
	// "syscall"
	"time"

	// "samuelemusiani/sasso/server/api"
	"samuelemusiani/sasso/server/config"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/dns"
	// "samuelemusiani/sasso/server/notify"
	// "samuelemusiani/sasso/server/proxmox"
)

var frontFS embed.FS

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

	// if c.Secrets.Key != "" {
	// 	slog.Info("Using secrets key provided in config file")
	// } else if c.Secrets.Path != "" {
	// 	slog.Debug("Trying to load secrets key from file", "path", c.Secrets.Path)
	// 	base64key, err := os.ReadFile(c.Secrets.Path)
	// 	if err != nil {
	// 		if !os.IsNotExist(err) {
	// 			slog.Error("Failed to read secrets key file", "error", err)
	// 			os.Exit(1)
	// 		}
	//
	// 		slog.Info("Secrets key file does not exist, generating new key", "path", c.Secrets.Path)
	// 		_, key, err := ed25519.GenerateKey(rand.Reader)
	// 		if err != nil {
	// 			slog.Error("Failed to generate new secrets key", "error", err)
	// 			os.Exit(1)
	// 		}
	//
	// 		base64key = []byte(base64.StdEncoding.EncodeToString(key))
	//
	// 		slog.Info("Saving key to file", "path", c.Secrets.Path)
	// 		err = os.WriteFile(c.Secrets.Path, base64key, 0600)
	// 		if err != nil {
	// 			slog.Error("Failed to write secrets key to file", "error", err)
	// 			os.Exit(1)
	// 		}
	//
	// 		c.Secrets.Key = string(base64key)
	// 	}
	// 	c.Secrets.Key = string(base64key)
	// } else {
	// 	slog.Error("No secrets key provided in config file or file path")
	// 	slog.Error("Please provide a secrets key in the config file or a path to a file containing the key")
	// 	os.Exit(1)
	// }
	//
	// real_key, err := base64.StdEncoding.DecodeString(c.Secrets.Key)
	// if err != nil {
	// 	slog.Error("Failed to decode secrets key", "error", err)
	// 	os.Exit(1)
	// }
	//
	// frontFS, err := fs.Sub(frontFS, "_front")
	// if err != nil {
	// 	slog.Error("Initializing change base path for front fs", "err", err)
	// 	os.Exit(1)
	// }

	// Database
	slog.Debug("Initializing database")
	dbLogger := slog.With("module", "db")
	err = db.Init(dbLogger, c.Database)
	if err != nil {
		slog.With("error", err).Error("Failed to initialize database")
		os.Exit(1)
	}

	// // Proxmox
	// slog.Debug("Initializing proxmox module")
	// proxmoxLogger := slog.With("module", "proxmox")
	// err = proxmox.Init(proxmoxLogger, c.Proxmox)
	// if err != nil {
	// 	slog.Error("Failed to initialize Proxmox client", "error", err)
	// 	os.Exit(1)
	// }
	//
	// ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	// defer cancel()
	//
	// slog.Debug("Starting background proxmox tasks")
	// go proxmox.TestEndpointVersion()
	// go proxmox.TestEndpointClone()
	// go proxmox.TestEndpointNetZone()
	// proxmox.StartWorker()
	//
	// // Notifications
	// notifyLogger := slog.With("module", "notify")
	// err = notify.Init(notifyLogger, c.Email)
	// if err != nil {
	// 	slog.Error("Failed to initialize notifications module", "error", err)
	// 	os.Exit(1)
	// }
	// notify.StartWorker()
	//
	// DNS
	dnsLogger := slog.With("module", "dns")
	err = dns.Init(dnsLogger, c.DNS)
	if err != nil {
		slog.Error("Failed to initialize DNS module", "error", err)
		os.Exit(1)
	}
	dns.StartWorker()

	for {
		time.Sleep(5 * time.Second)
	}

	// // API
	// slog.Debug("Initializing API server")
	// apiLogger := slog.With("module", "api")
	// api.Init(apiLogger, real_key, c.Secrets.InternalSecret, frontFS, c.PublicServer, c.PrivateServer)
	//
	// channelError := make(chan error, 1)
	//
	// go func() {
	// 	err = api.ListenAndServe()
	// 	if err != nil {
	// 		slog.Error("Failed to start API server", "error", err)
	// 	}
	// 	channelError <- err
	// }()
	//
	// select {
	// case err := <-channelError:
	// 	slog.Error("Server error", "error", err)
	// 	os.Exit(1)
	// case <-ctx.Done():
	// 	slog.Info("Received termination signal, shutting down...")
	// 	var waitGroup sync.WaitGroup
	// 	waitGroup.Add(4)
	//
	// 	go func() {
	// 		defer waitGroup.Done()
	// 		err := api.Shutdown()
	// 		if err != nil {
	// 			slog.Error("Failed to shut down API server", "error", err)
	// 		}
	// 	}()
	//
	// 	go func() {
	// 		defer waitGroup.Done()
	//
	// 		err = proxmox.ShutdownWorker()
	// 		if err != nil {
	// 			slog.Error("Failed to shut down Proxmox worker", "error", err)
	// 		}
	// 	}()
	//
	// 	go func() {
	// 		defer waitGroup.Done()
	//
	// 		err = notify.ShutdownWorker()
	// 		if err != nil {
	// 			slog.Error("Failed to shut down notifications worker", "error", err)
	// 		}
	// 	}()
	//
	// 	go func() {
	// 		defer waitGroup.Done()
	// 		err = dns.ShutdownWorker()
	// 		if err != nil {
	// 			slog.Error("Failed to shut down DNS worker", "error", err)
	// 		}
	// 	}()
	//
	// 	waitGroup.Wait()
	// }
	// slog.Info("Server shut down gracefully")
	// os.Exit(0)
}
