package proxmox

import (
	"context"
	"errors"
	"slices"

	"samuelemusiani/sasso/server/db"
	"time"
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

// This Function only creates a network in the database.
func AssignNewNetToUser(userID uint, name string, vlanaware bool) (*db.Net, error) {
	user, err := db.GetUserByID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get user by ID")
		return nil, err
	}

	nets, err := db.GetNetsByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get nets by user ID")
		return nil, err
	}

	if len(nets) >= int(user.MaxNets) {
		return nil, ErrInsufficientResources
	}

	if slices.IndexFunc(nets, func(n db.Net) bool { return n.Alias == name }) != -1 {
		logger.With("userID", userID, "name", name).Error("Network name already exists for user")
		return nil, ErrVNetNameExists
	}

	tag, err := db.GetRandomAvailableTagByZone(cNetwork.SDNZone, cNetwork.VXLANIDStart, cNetwork.VXLANIDEnd)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get available tag for creating network")
		return nil, err
	}

	if tag < cNetwork.VXLANIDStart || tag > cNetwork.VXLANIDEnd {
		logger.With("userID", userID, "tag", tag).Error("Tag is out of range")
		return nil, errors.New("Tag is out of range")
	}

	netName := cNetwork.SDNZone[0:3] + EncodeBase62(uint32(tag))

	net, err := db.CreateNetForUser(userID, netName, name, cNetwork.SDNZone, tag, vlanaware, string(VNetStatusPreCreating))
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to create network for user")
		return nil, err
	}

	return net, nil
}

func DeleteNet(userID uint, netID uint) error {
	net, err := db.GetNetByID(netID)
	if err != nil {
		logger.With("userID", userID, "netID", netID, "error", err).Error("Failed to get net by ID")
		return ErrVNetNotFound
	}

	interfaces, err := db.GetInterfacesByVNetID(netID)
	if err != nil {
		logger.With("userID", userID, "netID", netID, "error", err).Error("Failed to get interfaces by net ID")
		return err
	}
	if len(interfaces) > 0 {
		logger.With("userID", userID, "netID", netID).Error("Cannot delete net with active interfaces")
		return ErrVNetHasActiveInterfaces
	}

	if net.UserID != userID {
		logger.With("userID", userID, "netID", netID).Error("User is not the owner of the net")
		return ErrVNetNotFound
	}

	if err := db.UpdateVNetStatus(netID, string(VNetStatusPreDeleting)); err != nil {
		logger.With("userID", userID, "netID", netID, "error", err).Error("Failed to update net status to pre-deleting")
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
			logger.With("userID", userID, "vnetID", vnetID, "error", err).Error("Failed to get net by ID")
			return err
		}
	}
	if net.UserID != userID {
		return ErrVNetNotFound
	}

	nets, err := db.GetNetsByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get nets by user ID")
		return err
	}

	if slices.IndexFunc(nets, func(n db.Net) bool { return n.Alias == name && n.ID != vnetID }) != -1 {
		return ErrVNetNameExists
	}

	net.Alias = name
	if net.VlanAware != vlanware {
		// If vlanaware is changed, we need to set the status to reconfiguring
		// so that the worker will apply the change
		net.Status = string(VNetStatusReconfiguring)
		net.VlanAware = vlanware
	}

	if !vlanware {
		n, err := db.AreThereInterfacesWithVlanTagsByVNetID(vnetID)
		if err != nil {
			logger.With("userID", userID, "vnetID", vnetID, "error", err).Error("Failed to check for interfaces with vlan tags")
			return err
		}
		if n {
			return ErrVNetHasTaggedInterfaces
		}
	}

	if err := db.UpdateVNet(net); err != nil {
		logger.With("userID", userID, "vnetID", vnetID, "error", err).Error("Failed to update net")
		return err
	}

	return nil
}
