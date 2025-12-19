package fw

import (
	"errors"
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
		globalFirewall, err = NewShorewallFirewall(c.Shorewall)
		if err != nil {
			logger.Error("Failed to initialize Shorewall firewall", "error", err)
			return err
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
