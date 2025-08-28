package fw

import (
	"samuelemusiani/sasso/router/gateway"

	shorewall "github.com/samuelemusiani/go-shorewall"
)

func AddInterface(inter *gateway.Interface) error {
	if err := shorewall.AddZone(shorewall.Zone{
		Name: inter.VNet,
		Type: "ip",
	}); err != nil {
		return err
	}

	if err := shorewall.AddInterface(shorewall.Interface{
		Zone: inter.VNet,
		Name: inter.FirewallInterfaceName,
	}); err != nil {
		return err
	}

	if err := shorewall.AddSnat(shorewall.Snat{
		Action:      "MASQUERADE",
		Source:      inter.Subnet,
		Destination: inter.FirewallInterfaceName,
	}); err != nil {
		return err
	}

	if err := shorewall.AddPolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "out",
		Policy:      "ACCEPT",
	}); err != nil {
		return err
	}

	if err := shorewall.AddPolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "all",
		Policy:      "DROP",
	}); err != nil {
		return err
	}

	return shorewall.Reload()
}

func DeleteInterface(inter *gateway.Interface) error {
	if err := shorewall.RemovePolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "all",
		Policy:      "DROP",
	}); err != nil {
		return err
	}

	if err := shorewall.RemovePolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "out",
		Policy:      "ACCEPT",
	}); err != nil {
		return err
	}

	if err := shorewall.RemoveSnat(shorewall.Snat{
		Action:      "MASQUERADE",
		Source:      inter.Subnet,
		Destination: inter.FirewallInterfaceName,
	}); err != nil {
		return err
	}

	if err := shorewall.RemoveInterfaceByZone(inter.VNet); err != nil {
		return err
	}

	if err := shorewall.RemoveZone(inter.VNet); err != nil {
		return err
	}

	return shorewall.Reload()
}
