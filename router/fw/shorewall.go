package fw

import (
	"errors"
	"fmt"

	"github.com/samuelemusiani/go-shorewall"
)

type ShorewallFirewall struct {
	ExternalZone string
	VMZone       string
}

func (s *ShorewallFirewall) AddPortForward(outPort, destPort uint16, destIP string) error {
	err := goshorewall.AddRule(goshorewall.Rule{
		Action:      "DNAT",
		Source:      s.ExternalZone,
		Destination: fmt.Sprintf("%s:%s:%d", s.VMZone, destIP, destPort),
		Protocol:    "tcp,udp",
		Sport:       fmt.Sprintf("%d", outPort),
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
		Sport:       fmt.Sprintf("%d", outPort),
	})
	if err != nil && !errors.Is(err, goshorewall.ErrRuleNotFound) {
		return err
	}
	return goshorewall.Reload()
}
