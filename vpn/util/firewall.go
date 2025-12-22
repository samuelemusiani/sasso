package util

import (
	"fmt"

	shorewall "github.com/samuelemusiani/go-shorewall"
	"samuelemusiani/sasso/vpn/config"
)

func CreateRule(fwConfig config.Firewall, action string, peerAddress string, subnetSubnet string) shorewall.Rule {
	return shorewall.Rule{
		Action:      action,
		Source:      fmt.Sprintf("%s:%s", fwConfig.VPNZone, peerAddress),
		Destination: fmt.Sprintf("%s:%s", fwConfig.SassoZone, subnetSubnet),
	}
}
