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

	ErrUnsupportedGatewayType = errors.New("unsupported gateway type")
)

type Interface struct {
	// Global unique ID for this interface
	ID uint
	// Local ID on the gateway (e.g., if the gateway is Proxmox this is the ID of the interface on Proxmox net0, net1, etc)
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
	case "proxmox":
		pg := NewProxmoxGateway()

		err := pg.Init(c)
		if err != nil {
			logger.Error("Failed to initialize Proxmox gateway", "error", err)
			return err
		}

		globalGateway = pg
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
	Init(c config.Gateway) error
	NewInterface(vnet string, vnetID uint32, subnet, routerIP, broadcast string) (*Interface, error)
	RemoveInterface(id uint) error
	VerifyInterface(dbIface *Interface) (bool, error)
}

func (i *Interface) SaveToDB() error {
	var err error

	inter, err := db.GetInterfaceByVNet(i.VNet)
	switch {
	case err != nil && !errors.Is(err, db.ErrNotFound):
		logger.Error("Failed to get interface from database", "error", err, "vnet", i.VNet)
	case inter != nil:
		logger.Debug("Interface already exists in database", "vnet", i.VNet)

		inter.LocalID = i.LocalID
		inter.VNetID = i.VNetID
		inter.Subnet = i.Subnet
		inter.RouterIP = i.RouterIP
		inter.Broadcast = i.Broadcast
		inter.FirewallInterfaceName = i.FirewallInterfaceName

		err = db.UpdateInterface(*inter)
	default:
		err = db.SaveInterface(db.Interface{
			LocalID: i.LocalID,
			VNet:    i.VNet,
			VNetID:  i.VNetID,

			Subnet:    i.Subnet,
			RouterIP:  i.RouterIP,
			Broadcast: i.Broadcast,

			FirewallInterfaceName: i.FirewallInterfaceName,
		})
	}

	return err
}

func InterfaceFromDB(dbIface *db.Interface) *Interface {
	if dbIface == nil {
		return nil
	}

	return &Interface{
		ID:                    dbIface.ID,
		LocalID:               dbIface.LocalID,
		VNet:                  dbIface.VNet,
		VNetID:                dbIface.VNetID,
		Subnet:                dbIface.Subnet,
		RouterIP:              dbIface.RouterIP,
		Broadcast:             dbIface.Broadcast,
		FirewallInterfaceName: dbIface.FirewallInterfaceName,
	}
}
