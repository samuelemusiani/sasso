package fw

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	goshorewall "github.com/samuelemusiani/go-shorewall"
	"samuelemusiani/sasso/router/config"
)

type ShorewallFirewall struct {
	app *goshorewall.App

	externalZone string
	vmZone       string
	publicIP     string
}

func NewShorewallFirewall(c config.ShorewallFirewallConfig) (*ShorewallFirewall, error) {
	if c.ExternalZone == "" {
		return nil, errors.New("external zone cannot be empty")
	}

	if c.VMZone == "" {
		return nil, errors.New("VM zone cannot be empty")
	}

	if c.PublicIP == "" {
		return nil, errors.New("public IP cannot be empty")
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

	// TODO: We could check for ExternalZone and VMZone existence. But we cannot
	// do it with app.Zones() because it returns only the zones used by the app,
	// while those zones are global. If we use shorewall.Zones() instead we
	// could specify a custom base path. Wait for new go-shorewall version
	// https://github.com/samuelemusiani/go-shorewall/issues/4

	return &ShorewallFirewall{
		externalZone: c.ExternalZone,
		vmZone:       c.VMZone,
		publicIP:     c.PublicIP,
	}, nil
}

func (*ShorewallFirewall) ConstructPortForwardRule(outPort, destPort uint16, destIP string) Rule {
	return Rule{
		OutPort:  outPort,
		DestPort: destPort,
		DestIP:   destIP,
	}
}

func (s *ShorewallFirewall) PortForwardRules() ([]Rule, error) {
	srules, err := s.app.Rules()
	if err != nil {
		return nil, fmt.Errorf("failed to get shorewall rules: %w", err)
	}

	rules := make([]Rule, 0, len(srules))
	for _, sr := range srules {
		dest := strings.Split(sr.Destination, ":")
		if len(dest) != 3 {
			return nil, fmt.Errorf("unexpected destination format in shorewall rule: %s", sr.Destination)
		}

		var r Rule

		r.DestIP = dest[1]

		tmpPort, err := strconv.ParseUint(dest[2], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse destination port in shorewall rule: %w", err)
		}

		r.DestPort = uint16(tmpPort)

		tmpPort, err = strconv.ParseUint(sr.Dport, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse output port in shorewall rule: %w", err)
		}

		r.OutPort = uint16(tmpPort)

		rules = append(rules, r)
	}

	return rules, nil
}

func (s *ShorewallFirewall) shorewallRulefromRule(r Rule) goshorewall.Rule {
	return goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.externalZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.vmZone, r.DestIP, r.DestPort),
		Protocol:    "tcp,udp",
		Dport:       strconv.FormatUint(uint64(r.OutPort), 10),
	}
}

func (s *ShorewallFirewall) shorewallRulefromRuleNatReflection(r Rule) goshorewall.Rule {
	return goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.vmZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.vmZone, r.DestIP, r.DestPort),
		Protocol:    "tcp,udp",
		Dport:       strconv.FormatUint(uint64(r.OutPort), 10),
		Origdest:    s.publicIP,
	}
}

func (s *ShorewallFirewall) AddPortForwardRule(r Rule) error {
	reload := false

	err := s.app.AddRule(s.shorewallRulefromRule(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
			return err
		}
	} else {
		reload = true
	}

	// This rule is needed to have NAT reflection and allowing VMs from other
	// networks to access the forwarded ports using the public IP of the router
	err = s.app.AddRule(s.shorewallRulefromRuleNatReflection(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
			return err
		}
	} else {
		reload = true
	}

	if reload {
		return s.app.Reload()
	}

	return nil
}

func (s *ShorewallFirewall) AddPortForwardRules(rules []Rule) error {
	reload := false

	for i, r := range rules {
		err := s.app.AddRule(s.shorewallRulefromRule(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
				return errors.Join(err, fmt.Errorf("failed to add rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}

		// This rule is needed to have NAT reflection and allowing VMs from other
		// networks to access the forwarded ports using the public IP of the router
		err = s.app.AddRule(s.shorewallRulefromRuleNatReflection(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
				return errors.Join(err, fmt.Errorf("failed to add rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}
	}

	if reload {
		return s.app.Reload()
	}

	return nil
}

func (s *ShorewallFirewall) RemovePortForwardRule(r Rule) error {
	reload := false

	err := s.app.RemoveRule(s.shorewallRulefromRule(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleNotFound) {
			return err
		}
	} else {
		reload = true
	}

	// This rule is needed to have NAT reflection and allowing VMs from other
	// networks to access the forwarded ports using the public IP of the router
	err = s.app.RemoveRule(s.shorewallRulefromRuleNatReflection(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleNotFound) {
			return err
		}
	} else {
		reload = true
	}

	if reload {
		return s.app.Reload()
	}

	return nil
}

func (s *ShorewallFirewall) RemovePortForwardRules(rules []Rule) error {
	reload := false

	for i, r := range rules {
		err := s.app.RemoveRule(s.shorewallRulefromRule(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleNotFound) {
				return errors.Join(err, fmt.Errorf("failed to remove rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}

		// This rule is needed to have NAT reflection and allowing VMs from other
		// networks to access the forwarded ports using the public IP of the router
		err = s.app.RemoveRule(s.shorewallRulefromRuleNatReflection(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleNotFound) {
				return errors.Join(err, fmt.Errorf("failed to remove rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}
	}

	if reload {
		return s.app.Reload()
	}

	return nil
}

func sortRules(rules []goshorewall.Rule) []goshorewall.Rule {
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Compare(rules[j]) < 0
	})

	return rules
}

func searchSortedRules(r goshorewall.Rule, sortedRules []goshorewall.Rule) int {
	return sort.Search(len(sortedRules), func(i int) bool {
		return sortedRules[i].Compare(r) >= 0
	})
}

func (s *ShorewallFirewall) VerifyPortForwardRule(r Rule) (bool, error) {
	srules, err := s.app.Rules()
	if err != nil {
		logger.With("error", err).Error("Failed to get firewall rules")

		return false, err
	}

	sr1 := s.shorewallRulefromRule(r)

	sr2 := s.shorewallRulefromRuleNatReflection(r)
	if slices.Contains(srules, sr1) && slices.Contains(srules, sr2) {
		return true, nil
	}

	return false, nil
}

func (s *ShorewallFirewall) VerifyPortForwardRules(rules []Rule) ([]Rule, error) {
	srules, err := s.app.Rules()
	if err != nil {
		logger.With("error", err).Error("Failed to get firewall rules")

		return nil, err
	}

	srules = sortRules(srules)

	var faultyRules []Rule

	for _, r := range rules {
		sr1 := s.shorewallRulefromRule(r)
		sr2 := s.shorewallRulefromRuleNatReflection(r)
		i1 := searchSortedRules(sr1, srules)
		i2 := searchSortedRules(sr2, srules)

		if (i1 >= len(srules) || srules[i1].Compare(sr1) != 0) ||
			(i2 >= len(srules) || srules[i2].Compare(sr2) != 0) {
			// rule missing in firewall -> faulty
			faultyRules = append(faultyRules, r)
		}
	}

	return faultyRules, nil
}
