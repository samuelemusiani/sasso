package fw

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"

	goshorewall "github.com/samuelemusiani/go-shorewall"
	"samuelemusiani/sasso/router/config"
)

type ShorewallFirewall struct {
	ExternalZone string
	VMZone       string
	PublicIP     string
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

	v, err := goshorewall.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get shorewall version: %w", err)
	}

	logger.Info("Shorewall version", "version", v)

	zones, err := goshorewall.GetZones()
	if err != nil {
		return nil, fmt.Errorf("failed to get shorewall zones: %w", err)
	}

	fwZones := []string{c.ExternalZone, c.VMZone}
	for _, z := range fwZones {
		if !slices.ContainsFunc(zones, func(sz goshorewall.Zone) bool {
			return sz.Name == z
		}) {
			return nil, fmt.Errorf("shorewall zone %s not found", z)
		}
	}

	return &ShorewallFirewall{
		ExternalZone: c.ExternalZone,
		VMZone:       c.VMZone,
		PublicIP:     c.PublicIP,
	}, nil
}

func (*ShorewallFirewall) ConstructPortForwardRule(outPort, destPort uint16, destIP string) Rule {
	return Rule{
		OutPort:  outPort,
		DestPort: destPort,
		DestIP:   destIP,
	}
}

func (s *ShorewallFirewall) shorewallRulefromRule(r Rule) goshorewall.Rule {
	return goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.ExternalZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, r.DestIP, r.DestPort),
		Protocol:    "tcp,udp",
		Dport:       strconv.FormatUint(uint64(r.OutPort), 10),
	}
}

func (s *ShorewallFirewall) shorewallRulefromRuleNatReflection(r Rule) goshorewall.Rule {
	return goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.VMZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, r.DestIP, r.DestPort),
		Protocol:    "tcp,udp",
		Dport:       strconv.FormatUint(uint64(r.OutPort), 10),
		Origdest:    s.PublicIP,
	}
}

func (s *ShorewallFirewall) AddPortForwardRule(r Rule) error {
	reload := false

	err := goshorewall.AddRule(s.shorewallRulefromRule(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
			return err
		}
	} else {
		reload = true
	}

	// This rule is needed to have NAT reflection and allowing VMs from other
	// networks to access the forwarded ports using the public IP of the router
	err = goshorewall.AddRule(s.shorewallRulefromRuleNatReflection(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
			return err
		}
	} else {
		reload = true
	}

	if reload {
		return goshorewall.Reload()
	}

	return nil
}

func (s *ShorewallFirewall) AddPortForwardRules(rules []Rule) error {
	reload := false

	for i, r := range rules {
		err := goshorewall.AddRule(s.shorewallRulefromRule(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
				return errors.Join(err, fmt.Errorf("failed to add rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}

		// This rule is needed to have NAT reflection and allowing VMs from other
		// networks to access the forwarded ports using the public IP of the router
		err = goshorewall.AddRule(s.shorewallRulefromRuleNatReflection(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
				return errors.Join(err, fmt.Errorf("failed to add rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}
	}

	if reload {
		return goshorewall.Reload()
	}

	return nil
}

func (s *ShorewallFirewall) RemovePortForwardRule(r Rule) error {
	reload := false

	err := goshorewall.RemoveRule(s.shorewallRulefromRule(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleNotFound) {
			return err
		}
	} else {
		reload = true
	}

	// This rule is needed to have NAT reflection and allowing VMs from other
	// networks to access the forwarded ports using the public IP of the router
	err = goshorewall.RemoveRule(s.shorewallRulefromRuleNatReflection(r))
	if err != nil {
		if !errors.Is(err, goshorewall.ErrRuleNotFound) {
			return err
		}
	} else {
		reload = true
	}

	if reload {
		return goshorewall.Reload()
	}

	return nil
}

func (s *ShorewallFirewall) RemovePortForwardRules(rules []Rule) error {
	reload := false

	for i, r := range rules {
		err := goshorewall.RemoveRule(s.shorewallRulefromRule(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleNotFound) {
				return errors.Join(err, fmt.Errorf("failed to remove rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}

		// This rule is needed to have NAT reflection and allowing VMs from other
		// networks to access the forwarded ports using the public IP of the router
		err = goshorewall.RemoveRule(s.shorewallRulefromRuleNatReflection(r))
		if err != nil {
			if !errors.Is(err, goshorewall.ErrRuleNotFound) {
				return errors.Join(err, fmt.Errorf("failed to remove rule %d and subsequent rules", i))
			}
		} else {
			reload = true
		}
	}

	if reload {
		return goshorewall.Reload()
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
	srules, err := goshorewall.GetRules()
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
	srules, err := goshorewall.GetRules()
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
