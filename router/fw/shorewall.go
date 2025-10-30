package fw

import (
	"errors"
	"fmt"

	"github.com/samuelemusiani/go-shorewall"
)

type ShorewallFirewall struct {
	ExternalZone string
	VMZone       string
	PublicIP     string
}

func (s *ShorewallFirewall) AddPortForward(outPort, destPort uint16, destIP string) error {
	err := goshorewall.AddRule(goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.ExternalZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, destIP, destPort),
		Protocol:    "tcp,udp",
		Dport:       fmt.Sprintf("%d", outPort),
	})
	if err != nil && !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
		return err
	}

	// This rule is needed to have NAT reflection and allowing VMs from other
	// networks to access the forwarded ports using the public IP of the router
	err = goshorewall.AddRule(goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.VMZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, destIP, destPort),
		Protocol:    "tcp,udp",
		Dport:       fmt.Sprintf("%d", outPort),
		Origdest:    s.PublicIP,
	})
	if err != nil && !errors.Is(err, goshorewall.ErrRuleAlreadyExists) {
		return err
	}

	return goshorewall.Reload()
}

func (s *ShorewallFirewall) RemovePortForward(outPort, destPort uint16, destIP string) error {
	err := goshorewall.RemoveRule(goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.ExternalZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, destIP, destPort),
		Protocol:    "tcp,udp",
		Dport:       fmt.Sprintf("%d", outPort),
	})
	if err != nil && !errors.Is(err, goshorewall.ErrRuleNotFound) {
		return err
	}

	// This rule is needed to have NAT reflection and allowing VMs from other
	// networks to access the forwarded ports using the public IP of the router
	err = goshorewall.RemoveRule(goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.VMZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, destIP, destPort),
		Protocol:    "tcp,udp",
		Dport:       fmt.Sprintf("%d", outPort),
		Origdest:    s.PublicIP,
	})
	if err != nil && !errors.Is(err, goshorewall.ErrRuleNotFound) {
		return err
	}
	return goshorewall.Reload()
}
