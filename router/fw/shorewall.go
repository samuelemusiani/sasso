package fw

import (
	"errors"
	"fmt"

	goshorewall "github.com/samuelemusiani/go-shorewall"
)

type ShorewallFirewall struct {
	ExternalZone string
	VMZone       string
}

func (s *ShorewallFirewall) CreatePortForwardsRule(outPort, destPort uint16, destIP string) goshorewall.Rule {
	return goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.ExternalZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, destIP, destPort),
		Protocol:    "tcp,udp",
		Dport:       fmt.Sprintf("%d", outPort),
	}
}

func (s *ShorewallFirewall) AddPortForward(outPort, destPort uint16, destIP string) error {
	err := goshorewall.AddRule(s.CreatePortForwardsRule(outPort, destPort, destIP))
	if err != nil && !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
		return err
	}
	return goshorewall.Reload()
}

func (s *ShorewallFirewall) RemovePortForward(outPort, destPort uint16, destIP string) error {
	err := goshorewall.RemoveRule(s.CreatePortForwardsRule(outPort, destPort, destIP))
	if err != nil && !errors.Is(err, goshorewall.ErrRuleNotFound) {
		return err
	}
	return goshorewall.Reload()
}
