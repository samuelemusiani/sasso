package proxmox

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"samuelemusiani/sasso/server/config"

	"github.com/luthermonson/go-proxmox"
)

var (
	client *proxmox.Client = nil
	logger *slog.Logger    = nil
)

func Init(proxmoxLogger *slog.Logger, config config.Proxmox) error {
	logger = proxmoxLogger

	url := config.Url
	if !strings.Contains(config.Url, "api2/json") {
		if !strings.HasSuffix(config.Url, "/") {
			url += "/"
		}
		url += "api2/json"
	}

	http_client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.InsecureSkipVerify,
			},
		},
	}

	client = proxmox.NewClient(url,
		proxmox.WithHTTPClient(&http_client),
		proxmox.WithAPIToken(config.TokenID, config.Secret))

	return nil
}

func TestEndpointVersion() {
	first := true
	wasError := false

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		version, err := client.Version(ctx)
		cancel() // Cancel immediately after the call

		if err != nil {
			logger.Error("Failed to get Proxmox version", "error", err)
			wasError = true
		} else if first {
			logger.Info("Proxmox version", "version", version.Version)
			first = false
		} else if wasError {
			logger.Info("Proxmox version endpoint is back online", "version", version.Version)
			wasError = false
		}

		time.Sleep(10 * time.Second)
	}
}
