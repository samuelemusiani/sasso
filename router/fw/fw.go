package fw

import (
	"samuelemusiani/sasso/router/gateway"

	shorewall "github.com/samuelemusiani/go-shorewall"
)

func AddInterface(inter *gateway.Interface) error {
	if err := shorewall.AddZone(shorewall.Zone{
		Name: inter.VNet,
		Type: "ip",
	}); err != nil && err != shorewall.ErrZoneAlreadyExists {
		return err
	}

	if err := shorewall.AddInterface(shorewall.Interface{
		Zone: inter.VNet,
		Name: inter.FirewallInterfaceName,
	}); err != nil && err != shorewall.ErrInterfaceAlreadyExists {
		return err
	}

	if err := shorewall.AddPolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "out",
		Policy:      "ACCEPT",
	}); err != nil && err != shorewall.ErrPolicyAlreadyExists {
		return err
	}

	if err := shorewall.AddPolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "all",
		Policy:      "DROP",
	}); err != nil && err != shorewall.ErrPolicyAlreadyExists {
		return err
	}

	return shorewall.Reload()
}

func DeleteInterface(inter *gateway.Interface) error {
	if err := shorewall.RemovePolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "all",
		Policy:      "DROP",
	}); err != nil && err != shorewall.ErrPolicyNotFound {
		return err
	}

	if err := shorewall.RemovePolicy(shorewall.Policy{
		Source:      inter.VNet,
		Destination: "out",
		Policy:      "ACCEPT",
	}); err != nil && err != shorewall.ErrPolicyNotFound {
		return err
	}

	if err := shorewall.RemoveInterfaceByZone(inter.VNet); err != nil && err != shorewall.ErrInterfaceNotFound {
		return err
	}

	if err := shorewall.RemoveZone(inter.VNet); err != nil && err != shorewall.ErrZoneNotFound {
		return err
	}

	return shorewall.Reload()
}
