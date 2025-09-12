package proxmox

import (
	"context"
	"crypto/tls"
	"errors"
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

	cTemplate *config.ProxmoxTemplate = nil
	cClone    *config.ProxmoxClone    = nil
	cNetwork  *config.ProxmoxNetwork  = nil
	cGateway  *config.Gateway         = nil
	cVPN      *config.VPN             = nil

	ErrInvalidCloneIDTemplate = errors.New("invalid_clone_id_template")
	ErrInvalidSDNZone         = errors.New("invalid_sdn_zone")
	ErrInvalidVXLANRange      = errors.New("invalid_vxlan_range")
	ErrInsufficientResources  = errors.New("insufficient_resources")
	ErrTaskFailed             = errors.New("task_failed")

	xmoxReachable      = true
	isGatewayReachable = true
	isVPNReachable     = true
)

func Init(proxmoxLogger *slog.Logger, config config.Proxmox, gtwConfig config.Gateway, vpnConfig config.VPN) error {
	err := configChecks(config)
	if err != nil {
		return err
	}

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

	cTemplate = &config.Template
	cClone = &config.Clone
	cNetwork = &config.Network
	cGateway = &gtwConfig
	cVPN = &vpnConfig

	return nil
}

func configChecks(config config.Proxmox) error {
	idTemplate := strings.TrimSpace(config.Clone.IDTemplate)
	if !strings.Contains(idTemplate, "{{vmid}}") {
		logger.Error("Invalid Proxmox clone ID template. It must contain exaclty '{{vmid}}'", "template", idTemplate)
		return ErrInvalidCloneIDTemplate
	}

	tmp := len(strings.Replace(idTemplate, "{{vmid}}", "", 1)) + config.Clone.VMIDUserDigits + config.Clone.VMIDVMDigits
	if tmp < 3 || tmp > 9 {
		logger.Error("Invalid Proxmox clone ID template. The total length must be between 3 and 9 characters", "template", idTemplate, "length", tmp)
		return ErrInvalidCloneIDTemplate
	}

	if config.Clone.VMIDUserDigits < 1 || config.Clone.VMIDVMDigits < 1 {
		logger.Error("Invalid Proxmox clone ID template. The user digits and VM digits must be at least 1", "user_digits", config.Clone.VMIDUserDigits, "vm_digits", config.Clone.VMIDVMDigits)
		return ErrInvalidCloneIDTemplate
	}

	if config.Network.SDNZone == "" {
		logger.Error("Proxmox SDN zone is not configured", "zone", config.Network.SDNZone)
		return ErrInvalidSDNZone
	}

	if config.Network.VXLANIDStart <= 0 {
		logger.Error("Proxmox VXLAN ID start must be greater than 0", "vxlan_id_start", config.Network.VXLANIDStart)
		return ErrInvalidVXLANRange
	}

	if config.Network.VXLANIDEnd <= config.Network.VXLANIDStart {
		logger.Error("Proxmox VXLAN ID end must be greater than VXLAN ID start", "vxlan_id_start", config.Network.VXLANIDStart, "vxlan_id_end", config.Network.VXLANIDEnd)
		return ErrInvalidVXLANRange
	}

	if config.Network.VXLANIDEnd >= 1<<24 {
		logger.Error("Proxmox VXLAN ID end must be less than 16777216 (2^24)", "vxlan_id_end", config.Network.VXLANIDEnd)
		return ErrInvalidVXLANRange
	}

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
			xmoxReachable = false
		} else if first {
			logger.Info("Proxmox version", "version", version.Version)
			first = false
			xmoxReachable = true
		} else if wasError {
			logger.Info("Proxmox version endpoint is back online", "version", version.Version)
			wasError = false
			xmoxReachable = true
		}

		time.Sleep(10 * time.Second)
	}
}

func TestEndpointGateway() {
	first := true
	wasError := false

	for {
		req, err := http.NewRequest("GET", cGateway.Server+"/api/ping", nil)
		if err != nil {
			logger.Error("Failed to create request to Sasso gateway", "error", err)
			wasError = true
			isGatewayReachable = false
			time.Sleep(10 * time.Second)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+cGateway.Secret)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		cancel()

		if err != nil {
			logger.Error("Failed to reach Sasso gateway", "error", err)
			wasError = true
			isGatewayReachable = false
		} else {
			if resp.StatusCode != http.StatusOK {
				logger.With("status", resp.StatusCode).Error("Sasso gateway returned non-OK status")
				wasError = true
				isGatewayReachable = false
			} else if first {
				logger.Info("Sasso gateway is reachable", "status", resp.StatusCode)
				first = false
				isGatewayReachable = true
			} else if wasError {
				logger.Info("Sasso gateway is back online", "status", resp.StatusCode)
				wasError = false
				isGatewayReachable = true
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func TestEndpointVPN() {
	first := true
	wasError := false

	for {
		req, err := http.NewRequest("GET", cVPN.Server+"/api/ping", nil)
		if err != nil {
			logger.Error("Failed to create request to Sasso VPN", "error", err)
			wasError = true
			isVPNReachable = false
			time.Sleep(10 * time.Second)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+cVPN.Secret)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		cancel()

		if err != nil {
			logger.Error("Failed to reach Sasso VPN", "error", err)
			wasError = true
			isVPNReachable = false
		} else {
			if resp.StatusCode != http.StatusOK {
				logger.With("status", resp.StatusCode).Error("Sasso VPN returned non-OK status")
				wasError = true
				isVPNReachable = false
			} else if first {
				logger.Info("Sasso VPN is reachable", "status", resp.StatusCode)
				first = false
				isVPNReachable = true
			} else if wasError {
				logger.Info("Sasso VPN is back online", "status", resp.StatusCode)
				wasError = false
				isVPNReachable = true
			}
		}
		time.Sleep(10 * time.Second)
	}
}
