package fw

import (
	"errors"
	"fmt"
	"slices"

	goshorewall "github.com/samuelemusiani/go-shorewall"
	"samuelemusiani/sasso/vpn/config"
)

type ShorewallFirewall struct {
	app *goshorewall.App

	vpnZone   string
	sassoZone string
}

func newShorewallFirewall(c config.ShorewallFirewallConfig) (*ShorewallFirewall, error) {
	if c.VPNZone == "" {
		return nil, errors.New("vpn zone cannot be empty")
	}

	if c.SassoZone == "" {
		return nil, errors.New("sasso zone cannot be empty")
	}

	if c.ID == "" {
		return nil, errors.New("shorewall ID cannot be empty")
	}

	var app *goshorewall.App

	var err error

	if c.BasePath == "" {
		logger.Warn("Shorewall base path not set, using default provided by library")

		app, err = goshorewall.AppFromID(c.ID)
	} else {
		app, err = goshorewall.AppFromIDAndBasePath(c.ID, c.BasePath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize shorewall app: %w", err)
	}

	v, err := app.Version()
	if err != nil {
		return nil, fmt.Errorf("failed to get shorewall version: %w", err)
	}

	logger.Info("Shorewall version", "version", v)

	// TODO: We could check for VPN and Sasso zone existence. But we cannot
	// do it with app.Zones() because it returns only the zones used by the app,
	// while those zones are global. If we use shorewall.Zones() instead we
	// could specify a custom base path. Wait for new go-shorewall version
	// https://github.com/samuelemusiani/go-shorewall/issues/4

	return &ShorewallFirewall{
		app: app,

		vpnZone:   c.VPNZone,
		sassoZone: c.SassoZone,
	}, nil
}

func (s *ShorewallFirewall) shorewallRulefromRule(r Rule) goshorewall.Rule {
	return goshorewall.Rule{
		Action:      "ACCEPT",
		Source:      fmt.Sprintf("%s:%s", s.vpnZone, r.SrcIP),
		Destination: fmt.Sprintf("%s:%s", s.sassoZone, r.DestSubnet),
	}
}

func (*ShorewallFirewall) CreateAllowRule(srcIP, destSubnet string) Rule {
	return Rule{
		SrcIP:      srcIP,
		DestSubnet: destSubnet,
	}
}

func (s *ShorewallFirewall) ApplyRules(rules []Rule) error {
	wantedRules := make([]goshorewall.Rule, len(rules))
	for i, r := range rules {
		wantedRules[i] = s.shorewallRulefromRule(r)
	}

	currentRules, err := s.app.Rules()
	if err != nil {
		return fmt.Errorf("failed to get current shorewall rules: %w", err)
	}

	var rulesToAdd, rulesToRemove []goshorewall.Rule

	for _, wr := range wantedRules {
		if slices.ContainsFunc(currentRules, wr.Equals) {
			continue
		}

		rulesToAdd = append(rulesToAdd, wr)
	}

	for _, cr := range currentRules {
		if slices.ContainsFunc(wantedRules, cr.Equals) {
			continue
		}

		rulesToRemove = append(rulesToRemove, cr)
	}

	for _, r := range rulesToAdd {
		if err := s.app.AddRule(r); err != nil {
			return fmt.Errorf("failed to add shorewall rule: %w", err)
		}
	}

	for _, r := range rulesToRemove {
		if err := s.app.RemoveRule(r); err != nil {
			return fmt.Errorf("failed to remove shorewall rule: %w", err)
		}
	}

	if len(rulesToAdd) > 0 || len(rulesToRemove) > 0 {
		err = s.app.Reload()
		if err != nil {
			return fmt.Errorf("failed to reload shorewall: %w", err)
		}
	}

	return nil
}
