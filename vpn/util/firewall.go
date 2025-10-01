package util

import (
	"fmt"
	"samuelemusiani/sasso/vpn/config"

	shorewall "github.com/samuelemusiani/go-shorewall"
)

func CreateRule(fwConfig config.Firewall, action string, peerAddress string, subnetSubnet string) shorewall.Rule {
	return shorewall.Rule{
		Action:      action,
		Source:      fmt.Sprintf("%s:%s", fwConfig.VPNZone, peerAddress),
		Destination: fmt.Sprintf("%s:%s", fwConfig.SassoZone, subnetSubnet),
	}
}
