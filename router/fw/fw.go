package fw

import (
	"errors"
	"fmt"
	"log/slog"

	"samuelemusiani/sasso/router/config"
)

var (
	logger         *slog.Logger
	globalFirewall Firewall

	ErrUnsupportedFirewallType = errors.New("unsupported firewall type")
)

type Rule struct {
	OutPort  uint16
	DestPort uint16
	DestIP   string
}

type Firewall interface {
	ConstructPortForwardRule(outPort, destPort uint16, destIP string) Rule

	PortForwardRules() ([]Rule, error)

	AddPortForwardRule(r Rule) error
	AddPortForwardRules(rules []Rule) error

	RemovePortForwardRule(r Rule) error
	RemovePortForwardRules(rules []Rule) error

	VerifyPortForwardRule(r Rule) (bool, error)
	VerifyPortForwardRules(rules []Rule) ([]Rule, error)
}

func Init(l *slog.Logger, c config.Firewall) error {
	logger = l

	var err error

	switch c.Type {
	case "shorewall":
		logger.Info("Initializing Shorewall firewall")

		globalFirewall, err = newShorewallFirewall(c.Shorewall)
		if err != nil {
			return fmt.Errorf("failed to initialize Shorewall firewall: %w", err)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFirewallType, c.Type)
	}

	return nil
}

func Get() Firewall {
	return globalFirewall
}
