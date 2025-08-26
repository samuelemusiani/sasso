// This package provides an interface to handle the creation and management of
// the networks interfaces on the gateway. Multiple implementation can be possible
// the default one is based on the gateway being a Proxmox VM itself.
package gateway

import (
	"errors"
	"log/slog"

	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
)

var (
	logger        *slog.Logger
	globalGateway Gateway

	ErrUnsupportedGatewayType = errors.New("Unsupported gateway type")
)

type Interface struct {
	// Global unique ID for this interface
	ID uint
	// Local ID on the gateway (e.g., if the gateway is Proxmox this is the ID of the interface on Proxmox)
	LocalID uint
	VNet    string
	VNetID  uint
}

func Init(l *slog.Logger, c config.Gateway) error {
	logger = l

	switch c.Type {
	case "proxmox":
		pg := NewProxmoxGateway()
		err := pg.Init(c)
		if err != nil {
			logger.With("error", err).Error("Failed to initialize Proxmox gateway")
			return err
		}
		globalGateway = pg
	default:
		logger.With("type", c.Type).Error("Unsupported gateway type")
		return ErrUnsupportedGatewayType
	}

	return nil
}

func Get() Gateway {
	return globalGateway
}

type Gateway interface {
	Init(c config.Gateway) error
	NewInterface(vnet string, vnetID uint, routerIP string) (*Interface, error)
	RemoveInterface(id uint) error
}

func (i *Interface) SaveToDB() error {
	return db.SaveInterface(db.Interface{
		LocalID: i.LocalID,
		VNet:    i.VNet,
		VNetID:  i.VNetID,
	})
}
