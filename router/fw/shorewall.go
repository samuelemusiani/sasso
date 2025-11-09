package fw

import (
	"errors"
	"fmt"
	"sort"

	goshorewall "github.com/samuelemusiani/go-shorewall"
)

type ShorewallFirewall struct {
	ExternalZone string
	VMZone       string
}

func (s *ShorewallFirewall) ConstructPortForwardRule(outPort, destPort uint16, destIP string) Rule {
	return Rule{
		OutPort:  outPort,
		DestPort: destPort,
		DestIP:   destIP,
	}
}

func (s *ShorewallFirewall) ShorewallRulefromRule(r Rule) goshorewall.Rule {
	return goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.ExternalZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, r.DestIP, r.DestPort),
		Protocol:    "tcp,udp",
		Dport:       fmt.Sprintf("%d", r.OutPort),
	}
}

func (s *ShorewallFirewall) AddPortForwardRule(r Rule) error {
	err := goshorewall.AddRule(s.ShorewallRulefromRule(r))
	if err != nil && !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
		return err
	}
	return goshorewall.Reload()
}

func (s *ShorewallFirewall) AddPortForwardRules(rules []Rule) error {
	reload := false
	for _, r := range rules {
		err := goshorewall.AddRule(s.ShorewallRulefromRule(r))
		if errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
			reload = true
		} else if err != nil {
			return err
		}
	}

	if reload {
		return goshorewall.Reload()
	}
	return nil
}

func (s *ShorewallFirewall) RemovePortForwardRule(r Rule) error {
	err := goshorewall.RemoveRule(s.ShorewallRulefromRule(r))
	if err != nil && !errors.Is(err, goshorewall.ErrRuleNotFound) {
		return err
	}
	return goshorewall.Reload()
}

func (s *ShorewallFirewall) RemovePortForwardRules(rules []Rule) error {
	reload := false
	for _, r := range rules {
		err := goshorewall.RemoveRule(s.ShorewallRulefromRule(r))
		if errors.Is(err, goshorewall.ErrRuleNotFound) {
			reload = true
		} else if err != nil {
			return err
		}
	}

	if reload {
		return goshorewall.Reload()
	}
	return nil
}

func shorewallRuleLess(i *goshorewall.Rule, j *goshorewall.Rule) bool {
	if i.Action != j.Action {
		return i.Action < j.Action
	}
	if i.Destination != j.Destination {
		return i.Destination < j.Destination
	}
	if i.Dport != j.Dport {
		return i.Dport < j.Dport
	}
	if i.Protocol != j.Protocol {
		return i.Protocol < j.Protocol
	}
	if i.Source != j.Source {
		return i.Source < j.Source
	}
	return i.Sport < j.Sport
}

func sortRules(rules []goshorewall.Rule) []goshorewall.Rule {
	sort.Slice(rules, func(i, j int) bool {
		return shorewallRuleLess(&rules[i], &rules[j])
	})

	return rules
}

func searchSortedRules(r goshorewall.Rule, sortedRules []goshorewall.Rule) int {
	return sort.Search(len(sortedRules), func(i int) bool {
		return !shorewallRuleLess(&sortedRules[i], &r) // >= is equal to not <
	})
}

func (s *ShorewallFirewall) VerifyPortForwardRule(r Rule) (bool, error) {
	srules, err := goshorewall.GetRules()
	if err != nil {
		logger.With("error", err).Error("Failed to get firewall rules")
		return false, err
	}

	sr := s.ShorewallRulefromRule(r)
	for _, srr := range srules {
		if srr == sr {
			// rule present in firewall -> ok
			return true, nil
		}
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
		sr := s.ShorewallRulefromRule(r)
		i := searchSortedRules(sr, srules)
		if i >= len(srules) || srules[i] != sr {
			// rule missing in firewall -> faulty
			faultyRules = append(faultyRules, r)
		}
	}

	return faultyRules, nil
}
