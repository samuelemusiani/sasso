package proxmox

import (
	"context"
	"errors"
	"slices"
	"time"

	"samuelemusiani/sasso/server/db"
)

var (
	VNetStatusUnknown       VMStatus = "unknown"
	VNetStatusPending       VMStatus = "pending"
	VNetStatusReady         VMStatus = "ready"
	VNetStatusReconfiguring VMStatus = "reconfiguring"

	// The pre-status is before the main worker has acknowledged the creation or
	// deletion
	VNetStatusPreCreating VMStatus = "pre-creating"
	VNetStatusPreDeleting VMStatus = "pre-deleting"

	// This status is then the main worker has taken an action, but the vm
	// is not yet fully cloned or deleted.
	VNetStatusCreating VMStatus = "creating"
	VNetStatusDeleting VMStatus = "deleting"

	ErrVNetNotFound            error = errors.New("VNet not found")
	ErrVNetHasActiveInterfaces error = errors.New("VNet has active interfaces")
	ErrVNetNameExists          error = errors.New("VNet name already exists")
	ErrVNetHasTaggedInterfaces error = errors.New("VNet has tagged interfaces")
)

func TestEndpointNetZone() {
	time.Sleep(5 * time.Second)

	wasError := false
	first := true

	for {
		if !isProxmoxReachable {
			time.Sleep(20 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cluster, err := client.Cluster(ctx)

		cancel()

		if err != nil {
			logger.Error("Failed to get Proxmox cluster", "error", err)
			time.Sleep(10 * time.Second)

			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		zone, err := cluster.SDNZone(ctx, cNetwork.SDNZone)

		cancel()

		if err != nil {
			logger.Error("Failed to get Proxmox SDN cluster zone", "error", err)

			wasError = true

			time.Sleep(10 * time.Second)

			continue
		}

		if zone.Name != cNetwork.SDNZone {
			logger.Error("Proxmox SDN cluster zone name mismatch", "expected", cNetwork.SDNZone, "got", zone.Name)

			wasError = true

			time.Sleep(10 * time.Second)

			continue
		}

		if zone.Type != "vxlan" {
			logger.Error("Proxmox SDN cluster zone type mismatch", "expected", "vxlan", "got", zone.Type)

			wasError = true

			time.Sleep(10 * time.Second)

			continue
		}

		if first {
			logger.Info("Proxmox SDN cluster zone is valid", "name", zone.Name, "type", zone.Type)

			first = false
		} else if wasError {
			logger.Info("Proxmox SDN cluster zone is valid again after error", "name", zone.Name, "type", zone.Type)

			wasError = false
		}

		// Should check the state or if it's pending?

		time.Sleep(10 * time.Second)
	}
}

// CreateNewNet only creates a network in the database.
func CreateNewNet(userID uint, name string, vlanaware bool, groupID *uint) (*db.Net, error) {
	user, err := db.GetUserByID(userID)
	if err != nil {
		logger.Error("Failed to get user by ID", "userID", userID, "error", err)
		return nil, err
	}

	// It's a net group
	if groupID != nil {
		role, err := db.GetUserRoleInGroup(userID, *groupID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				return nil, ErrNotFound
			}

			logger.Error("Failed to get user role in group", "userID", userID, "groupID", *groupID, "error", err)

			return nil, err
		}

		if role != "admin" && role != "owner" {
			return nil, ErrPermissionDenied
		}
	}

	var nets []db.Net

	if groupID != nil {
		_, _, _, maxNets, err := db.GetGroupResourceLimits(*groupID)
		if err != nil {
			logger.Error("Failed to get group resources by group ID", "groupID", *groupID, "error", err)
			return nil, err
		}

		nets, err = db.GetNetsByGroupID(*groupID)
		if err != nil {
			logger.Error("Failed to get nets by group ID", "groupID", *groupID, "error", err)
			return nil, err
		}

		if len(nets)+1 > int(maxNets) {
			return nil, ErrInsufficientResources
		}
	} else {
		nets, err = db.GetNetsByUserID(userID)
		if err != nil {
			logger.Error("Failed to get nets by user ID", "userID", userID, "error", err)
			return nil, err
		}

		if len(nets) >= int(user.MaxNets) {
			return nil, ErrInsufficientResources
		}
	}

	l := logger
	if groupID != nil {
		l = logger.With("groupID", *groupID)
	}

	if slices.IndexFunc(nets, func(n db.Net) bool { return n.Alias == name }) != -1 {
		l.Error("Network name already exists for user or group", "userID", userID, "name", name)
		return nil, ErrVNetNameExists
	}

	tag, err := db.GetRandomAvailableTagByZone(cNetwork.SDNZone, cNetwork.VXLANIDStart, cNetwork.VXLANIDEnd)
	if err != nil {
		logger.Error("Failed to get available tag for creating network", "userID", userID, "error", err)
		return nil, err
	}

	if tag < cNetwork.VXLANIDStart || tag > cNetwork.VXLANIDEnd {
		logger.Error("tag is out of range", "userID", userID, "tag", tag)
		return nil, errors.New("tag is out of range")
	}

	netName := cNetwork.SDNZone[0:3] + EncodeBase62(tag)

	var net *db.Net
	if groupID != nil {
		net, err = db.CreateNetForGroup(*groupID, netName, name, cNetwork.SDNZone, tag, vlanaware, string(VNetStatusPreCreating))
		if err != nil {
			logger.Error("Failed to create network for group", "groupID", *groupID, "error", err)
			return nil, err
		}
	} else {
		net, err = db.CreateNetForUser(userID, netName, name, cNetwork.SDNZone, tag, vlanaware, string(VNetStatusPreCreating))
		if err != nil {
			logger.Error("Failed to create network for user", "userID", userID, "error", err)
			return nil, err
		}
	}

	return net, nil
}

func DeleteNet(userID uint, netID uint) error {
	net, err := db.GetNetByID(netID)
	if err != nil {
		logger.Error("Failed to get net by ID", "userID", userID, "netID", netID, "error", err)
		return ErrVNetNotFound
	}

	interfaces, err := db.GetInterfacesByVNetID(netID)
	if err != nil {
		logger.Error("Failed to get interfaces by net ID", "userID", userID, "netID", netID, "error", err)
		return err
	}

	if len(interfaces) > 0 {
		logger.Error("Cannot delete net with active interfaces", "ownerID", userID, "netID", netID)
		return ErrVNetHasActiveInterfaces
	}

	switch net.OwnerType {
	case "User":
		if net.OwnerID != userID {
			return ErrVNetNotFound
		}
	case "Group":
		role, err := db.GetUserRoleInGroup(userID, net.OwnerID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				return ErrVNetNotFound
			}

			logger.Error("Failed to get user role in group", "userID", userID, "groupID", net.OwnerID, "netID", netID, "error", err)

			return err
		}

		if role != "admin" && role != "owner" {
			return ErrPermissionDenied
		}
	default:
		logger.Error("Invalid net owner type", "ownerID", userID, "netID", netID, "ownerType", net.OwnerType)
		return ErrVNetNotFound
	}

	if err := db.UpdateVNetStatus(netID, string(VNetStatusPreDeleting)); err != nil {
		logger.Error("Failed to update net status to pre-deleting", "userID", userID, "netID", netID, "error", err)
		return err
	}

	return nil
}

func UpdateNet(userID, vnetID uint, name string, vlanware bool) error {
	net, err := db.GetNetByID(vnetID)
	if err != nil {
		if err == db.ErrNotFound {
			return ErrVNetNotFound
		} else {
			logger.Error("Failed to get net by ID", "userID", userID, "vnetID", vnetID, "error", err)
			return err
		}
	}

	switch net.OwnerType {
	case "User":
		if net.OwnerID != userID {
			return ErrVNetNotFound
		}

	case "Group":
		role, err := db.GetUserRoleInGroup(userID, net.OwnerID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				return ErrVNetNotFound
			}

			logger.Error("Failed to get user role in group", "userID", userID, "groupID", net.OwnerID, "netID", vnetID, "error", err)

			return err
		}

		if role != "admin" && role != "owner" {
			return ErrPermissionDenied
		}
	default:
		logger.Error("Invalid net owner type", "ownerID", userID, "vnetID", vnetID, "ownerType", net.OwnerType)

		return ErrVNetNotFound
	}

	var nets []db.Net
	if net.OwnerType == "Group" {
		nets, err = db.GetNetsByGroupID(net.OwnerID)
		if err != nil {
			logger.Error("Failed to get nets by group ID", "groupID", net.OwnerID, "error", err)
			return err
		}
	} else {
		nets, err = db.GetNetsByUserID(userID)
		if err != nil {
			logger.Error("Failed to get nets by user ID", "userID", userID, "error", err)
			return err
		}
	}

	if slices.IndexFunc(nets, func(n db.Net) bool { return n.Alias == name && n.ID != vnetID }) != -1 {
		return ErrVNetNameExists
	}

	changed := false

	if name != "" {
		net.Alias = name
		changed = true
	}

	if net.VlanAware != vlanware {
		// If vlanaware is changed, we need to set the status to reconfiguring
		// so that the worker will apply the change
		net.Status = string(VNetStatusReconfiguring)
		net.VlanAware = vlanware
		changed = true
	}

	if !vlanware {
		n, err := db.AreThereInterfacesWithVlanTagsByVNetID(vnetID)
		if err != nil {
			logger.Error("Failed to check for interfaces with vlan tags", "userID", userID, "vnetID", vnetID, "error", err)
			return err
		}

		if n {
			return ErrVNetHasTaggedInterfaces
		}
	}

	if !changed {
		return nil
	}

	if err := db.UpdateVNet(net); err != nil {
		logger.Error("Failed to update net", "userID", userID, "vnetID", vnetID, "error", err)
		return err
	}

	return nil
}
