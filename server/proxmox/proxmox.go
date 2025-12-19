package proxmox

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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

	// nonce is used to generate backup names
	nonce []byte = nil

	ErrInvalidCloneIDTemplate = errors.New("invalid_clone_id_template")
	ErrInvalidSDNZone         = errors.New("invalid_sdn_zone")
	ErrInvalidVXLANRange      = errors.New("invalid_vxlan_range")
	ErrInsufficientResources  = errors.New("insufficient_resources")
	ErrTaskFailed             = errors.New("task_failed")
	ErrInvalidStorage         = errors.New("invalid_storage")
	ErrCantGenerateNonce      = errors.New("cant_generate_nonce")
	ErrPermissionDenied       = errors.New("permission_denied")
	ErrNotFound               = errors.New("a resouces can't be found")

	isProxmoxReachable = true
)

func Init(proxmoxLogger *slog.Logger, config config.Proxmox) error {
	logger = proxmoxLogger
	if err := checkConfig(&config); err != nil {
		return err
	}

	// Generate a nonce for backup names
	nonce = make([]byte, 32)
	n, err := rand.Read(nonce)
	if err != nil && n != 32 {
		logger.Error("Failed to generate random key for nonce", "error", err)
		return ErrCantGenerateNonce
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

func checkConfig(c *config.Proxmox) error {
	_, err := url.Parse(c.Url)
	if err != nil {
		return fmt.Errorf("invalid Proxmox URL: %w", err)
	}

	if c.TokenID == "" {
		return errors.New("Proxmox token ID is required")
	}
	if c.Secret == "" {
		return errors.New("Proxmox secret is required")
	}

	if c.Clone.TargetNode == "" {
		return errors.New("Proxmox clone target node is required")
	}

	idTemplate := strings.TrimSpace(c.Clone.IDTemplate)
	if !strings.Contains(idTemplate, "{{vmid}}") {
		e := fmt.Errorf("Invalid Proxmox clone ID template. It must contain exaclty '{{vmid}}'. template: %s", idTemplate)
		return errors.Join(ErrInvalidCloneIDTemplate, e)
	}

	tmp := len(strings.Replace(idTemplate, "{{vmid}}", "", 1)) + c.Clone.VMIDUserDigits + c.Clone.VMIDVMDigits
	if tmp < 3 || tmp > 9 {
		e := fmt.Errorf("Invalid Proxmox clone ID template. The total length must be between 3 and 9 characters. template: %s length: %d", idTemplate, tmp)
		return errors.Join(ErrInvalidCloneIDTemplate, e)
	}

	if c.Clone.VMIDUserDigits < 1 || c.Clone.VMIDVMDigits < 1 {
		e := fmt.Errorf("Invalid Proxmox clone ID template. The user digits and VM digits must be at least 1. user_digits: %d vm_digits: %d", c.Clone.VMIDUserDigits, c.Clone.VMIDVMDigits)
		return errors.Join(ErrInvalidCloneIDTemplate, e)
	}

	if c.Clone.MTU.MTU == 0 && c.Clone.MTU.Set {
		return errors.New("invalid_proxmox_clone_mtu")
	}

	if c.Template.Node == "" {
		return errors.New("Proxmox template node is required")
	}

	if c.Template.VMID == 0 {
		return errors.New("Proxmox template VMID is required")
	} else if c.Template.VMID < 100 {
		return errors.New("Proxmox template VMID must be greater than or equal to 100")
	}

	if c.Network.SDNZone == "" {
		e := fmt.Errorf("Proxmox SDN zone is not configured. zone: %s", c.Network.SDNZone)
		return errors.Join(ErrInvalidSDNZone, e)
	}

	if c.Network.VXLANIDStart <= 0 {
		e := fmt.Errorf("Proxmox VXLAN ID start must be greater than 0. vxlan_id_start%d", c.Network.VXLANIDStart)
		return errors.Join(ErrInvalidVXLANRange, e)
	}

	if c.Network.VXLANIDEnd <= c.Network.VXLANIDStart {
		e := fmt.Errorf("Proxmox VXLAN ID end must be greater than VXLAN ID start. vxlan_id_start: %d. vxlan_id_end: %d", c.Network.VXLANIDStart, c.Network.VXLANIDEnd)
		return errors.Join(ErrInvalidVXLANRange, e)
	}

	if c.Network.VXLANIDEnd >= 1<<24 {
		e := fmt.Errorf("Proxmox VXLAN ID end must be less than 16777216 (2^24). vxlan_id_end: %d", c.Network.VXLANIDEnd)
		return errors.Join(ErrInvalidVXLANRange, e)
	}

	if c.Backup.Storage == "" {
		e := errors.New("Proxmox backup storage is not configured")
		return errors.Join(ErrInvalidStorage, e)
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
