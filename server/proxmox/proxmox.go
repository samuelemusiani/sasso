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

	"github.com/luthermonson/go-proxmox"
	"samuelemusiani/sasso/server/config"
)

var (
	client *proxmox.Client
	logger *slog.Logger

	cTemplate *config.ProxmoxTemplate
	cClone    *config.ProxmoxClone
	cNetwork  *config.ProxmoxNetwork
	cBackup   *config.ProxmoxBackup

	// nonce is used to generate backup names
	nonce []byte

	ErrInvalidCloneIDTemplate = errors.New("invalid clone id template")
	ErrInvalidSDNZone         = errors.New("invalid sdn zone")
	ErrInvalidVXLANRange      = errors.New("invalid vxlan range")
	ErrInsufficientResources  = errors.New("insufficient resources")
	ErrTaskFailed             = errors.New("task failed")
	ErrInvalidStorage         = errors.New("invalid storage")
	ErrCantGenerateNonce      = errors.New("cannot generate nonce")
	ErrPermissionDenied       = errors.New("permission denied")
	ErrNotFound               = errors.New("a resouces can't be found")
	ErrUnsupportedOwnerType   = errors.New("unsupported owner type")

	isProxmoxReachable = true
)

// OwnerType represents the type of owner for a resource.
type OwnerType int

const (
	// OwnerTypeUser indicates that the owner is a user and the OwnerID refers
	// to a user ID.
	OwnerTypeUser OwnerType = iota
	// OwnerTypeGroup indicates that the owner is a group and the OwnerID refers
	// to a group ID.
	OwnerTypeGroup
)

func Init(proxmoxLogger *slog.Logger, c config.Proxmox) error {
	logger = proxmoxLogger

	if err := checkConfig(&c); err != nil {
		return err
	}

	// Generate a nonce for backup names
	nonce = make([]byte, 32)

	n, err := rand.Read(nonce)
	if err != nil && n != 32 {
		logger.Error("Failed to generate random key for nonce", "error", err)

		return ErrCantGenerateNonce
	}

	proxmoxURL := c.URL
	if !strings.Contains(c.URL, "api2/json") {
		if !strings.HasSuffix(c.URL, "/") {
			proxmoxURL += "/"
		}

		proxmoxURL += "api2/json"
	}

	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.InsecureSkipVerify,
			},
		},
	}

	client = proxmox.NewClient(proxmoxURL,
		proxmox.WithHTTPClient(&httpClient),
		proxmox.WithAPIToken(c.TokenID, c.Secret))

	cTemplate = &c.Template
	cClone = &c.Clone
	cNetwork = &c.Network
	cBackup = &c.Backup

	return nil
}

func checkConfig(c *config.Proxmox) error {
	_, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf("invalid Proxmox URL: %w", err)
	}

	if c.TokenID == "" {
		return errors.New("proxmox token ID is required")
	}

	if c.Secret == "" {
		return errors.New("proxmox secret is required")
	}

	if c.Clone.TargetNode == "" {
		return errors.New("proxmox clone target node is required")
	}

	idTemplate := strings.TrimSpace(c.Clone.IDTemplate)
	if !strings.Contains(idTemplate, "{{vmid}}") {
		e := fmt.Errorf("invalid Proxmox clone ID template. It must contain exactly '{{vmid}}'. template: %s", idTemplate)

		return errors.Join(ErrInvalidCloneIDTemplate, e)
	}

	tmp := len(strings.Replace(idTemplate, "{{vmid}}", "", 1)) + c.Clone.VMIDUserDigits + c.Clone.VMIDVMDigits
	if tmp < 3 || tmp > 9 {
		e := fmt.Errorf("invalid Proxmox clone ID template. The total length must be between 3 and 9 characters. template: %s length: %d", idTemplate, tmp)

		return errors.Join(ErrInvalidCloneIDTemplate, e)
	}

	if c.Clone.VMIDUserDigits < 1 || c.Clone.VMIDVMDigits < 1 {
		e := fmt.Errorf("invalid Proxmox clone ID template. The user digits and VM digits must be at least 1. user_digits: %d vm_digits: %d", c.Clone.VMIDUserDigits, c.Clone.VMIDVMDigits)

		return errors.Join(ErrInvalidCloneIDTemplate, e)
	}

	if c.Clone.MTU.MTU == 0 && c.Clone.MTU.Set {
		return errors.New("invalid_proxmox_clone_mtu")
	}

	if c.Template.Node == "" {
		return errors.New("proxmox template node is required")
	}

	if c.Template.VMID == 0 {
		return errors.New("proxmox template VMID is required")
	} else if c.Template.VMID < 100 {
		return errors.New("proxmox template VMID must be greater than or equal to 100")
	}

	if c.Network.SDNZone == "" {
		e := fmt.Errorf("proxmox SDN zone is not configured. zone: %s", c.Network.SDNZone)

		return errors.Join(ErrInvalidSDNZone, e)
	}

	if c.Network.VXLANIDStart <= 0 {
		e := fmt.Errorf("proxmox VXLAN ID start must be greater than 0. vxlan_id_start%d", c.Network.VXLANIDStart)

		return errors.Join(ErrInvalidVXLANRange, e)
	}

	if c.Network.VXLANIDEnd <= c.Network.VXLANIDStart {
		e := fmt.Errorf("proxmox VXLAN ID end must be greater than VXLAN ID start. vxlan_id_start: %d. vxlan_id_end: %d", c.Network.VXLANIDStart, c.Network.VXLANIDEnd)

		return errors.Join(ErrInvalidVXLANRange, e)
	}

	if c.Network.VXLANIDEnd >= 1<<24 {
		e := fmt.Errorf("proxmox VXLAN ID end must be less than 16777216 (2^24). vxlan_id_end: %d", c.Network.VXLANIDEnd)

		return errors.Join(ErrInvalidVXLANRange, e)
	}

	if c.Backup.Storage == "" {
		e := errors.New("proxmox backup storage is not configured")

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

		switch {
		case err != nil:
			logger.Error("Failed to get Proxmox version", "error", err)

			wasError = true
			isProxmoxReachable = false
		case first:
			logger.Info("proxmox version", "version", version.Version)

			first = false
			isProxmoxReachable = true
		case wasError:
			logger.Info("proxmox version endpoint is back online", "version", version.Version)

			wasError = false
			isProxmoxReachable = true
		}

		time.Sleep(10 * time.Second)
	}
}
