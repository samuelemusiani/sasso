package fw

import (
	"errors"
	"fmt"
	"log/slog"

	"samuelemusiani/sasso/vpn/config"
)

var (
	logger *slog.Logger

	ErrUnsupportedFirewallType = errors.New("unsupported firewall type")
)

type Rule struct {
	SrcIP      string
	DestSubnet string
}

func ConstructAllowRule(srcIP, destSubnet string) Rule {
	return Rule{
		SrcIP:      srcIP,
		DestSubnet: destSubnet,
	}
}

type Firewall interface {
	CreateAllowRule(srcIP, destSubnet string) Rule
	ApplyRules(rules []Rule) error
}

func Init(l *slog.Logger, c config.Firewall) (Firewall, error) {
	logger = l

	var err error

	var firewall Firewall

	switch c.Type {
	case "shorewall":
		logger.Info("Initializing Shorewall firewall")

		firewall, err = newShorewallFirewall(c.Shorewall)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Shorewall firewall: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFirewallType, c.Type)
	}

	return firewall, nil
}
