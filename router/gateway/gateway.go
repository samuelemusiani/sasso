// This package provides an interface to handle the creation and management of
// the networks interfaces on the Router. Multiple implementation can be
// possible.
package gateway

import (
	"errors"
	"log/slog"

	"samuelemusiani/sasso/router/config"
)

var (
	logger        *slog.Logger
	globalGateway Gateway

	ErrUnsupportedGatewayType = errors.New("unsupported gateway type")
)

type Interface struct {
	// Local ID on the Router (usually it's the interface number as linux sees it)
	LocalID uint
	VNet    string
	VNetID  uint32

	Subnet    string
	RouterIP  string
	Broadcast string

	// Name of the interface on the gateway. enpXsY or ethX or similar
	FirewallInterfaceName string
}

func Init(l *slog.Logger, c config.Gateway) error {
	logger = l

	switch c.Type {
	case "linux":
		lg := NewLinuxGateway()

		err := lg.Init(c)
		if err != nil {
			logger.Error("Failed to initialize Linux gateway", "error", err)

			return err
		}

		globalGateway = lg
	default:
		logger.Error("Unsupported gateway type", "type", c.Type)

		return ErrUnsupportedGatewayType
	}

	return nil
}

func Get() Gateway {
	return globalGateway
}

type Gateway interface {
	// Initialize the gateway with the given configuration. This should be called
	// once before any other method.
	Init(c config.Gateway) error
	NewInterface(vnet string, vnetID uint32, subnet, routerIP, broadcast string) (*Interface, error)
	RemoveInterface(id uint) error
	// VerifyInterface checks if the given interface is correctly configured on the
	// gateway. It returns true if the interface is correctly configured, false
	// otherwise
	VerifyInterface(dbIface *Interface) (bool, error)
	// GetAllInterfaces returns all the interfaces configured on the gateway. This
	// can be used to check if the current applied status is consistent with
	// wanted status.
	GetAllInterfaces() ([]*Interface, error)
}
