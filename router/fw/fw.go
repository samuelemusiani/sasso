package fw

import (
	"errors"
	"log/slog"
	"samuelemusiani/sasso/router/config"

	goshorewall "github.com/samuelemusiani/go-shorewall"
)

var (
	logger         *slog.Logger
	globalFirewall Firewall

	ErrUnsupportedFirewallType = errors.New("unsupported firewall type")
)

type Firewall interface {
	CreatePortForwardsRule(outPort, destPort uint16, destIP string) goshorewall.Rule
	AddPortForward(outPort, destPort uint16, destIP string) error
	RemovePortForward(outPort, destPort uint16, destIP string) error
}

func Init(l *slog.Logger, c config.Firewall) error {
	logger = l

	switch c.Type {
	case "shorewall":
		logger.Info("Initializing Shorewall firewall")
		globalFirewall = &ShorewallFirewall{
			ExternalZone: c.Shorewall.ExternalZone,
			VMZone:       c.Shorewall.VMZone,
		}
	default:
		logger.Error("Unsupported firewall type", "type", c.Type)
		return ErrUnsupportedFirewallType
	}

	return nil
}

func Get() Firewall {
	return globalFirewall
}
