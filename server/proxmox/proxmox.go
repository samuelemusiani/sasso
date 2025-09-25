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
	cBackup   *config.ProxmoxBackup   = nil

	ErrInvalidCloneIDTemplate = errors.New("invalid_clone_id_template")
	ErrInvalidSDNZone         = errors.New("invalid_sdn_zone")
	ErrInvalidVXLANRange      = errors.New("invalid_vxlan_range")
	ErrInsufficientResources  = errors.New("insufficient_resources")
	ErrTaskFailed             = errors.New("task_failed")
	ErrInvalidStorage         = errors.New("invalid_storage")

	isProxmoxReachable = true
	isGatewayReachable = true
	isVPNReachable     = true
)

func Init(proxmoxLogger *slog.Logger, config config.Proxmox) error {
	logger = proxmoxLogger

	err := configChecks(config)
	if err != nil {
		return err
	}

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
	cBackup = &config.Backup

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

	if config.Backup.Storage == "" {
		logger.Error("Proxmox backup storage is not configured")
		return ErrInvalidStorage
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
			isProxmoxReachable = false
		} else if first {
			logger.Info("Proxmox version", "version", version.Version)
			first = false
			isProxmoxReachable = true
		} else if wasError {
			logger.Info("Proxmox version endpoint is back online", "version", version.Version)
			wasError = false
			isProxmoxReachable = true
		}

		time.Sleep(10 * time.Second)
	}
}
