package proxmox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"samuelemusiani/sasso/server/db"
	"time"
)

var (
	VNetStatusUnknown VMStatus = "unknown"
	VNetStatusPending VMStatus = "pending"
	VNetStatusReady   VMStatus = "ready"

	// The pre-status is before the main worker has acknowledged the creation or
	// deletion
	VNetStatusPreCreating VMStatus = "pre-creating"
	VNetStatusPreDeleting VMStatus = "pre-deleting"

	// This status is then the main worker has taken an action, but the vm
	// is not yet fully cloned or deleted.
	VNetStatusCreating VMStatus = "creating"
	VNetStatusDeleting VMStatus = "deleting"

	ErrVNetNotFound error = errors.New("VNet not found")
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
func AssignNewNetToUser(userID uint, name string) (*db.Net, error) {
	lastTag, err := db.GetLastUsedTagByZone(cNetwork.SDNZone)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get last used tag for creating network")
		return nil, err
	}

	newTag := lastTag + 1
	if newTag > cNetwork.VXLANIDEnd {
		logger.With("userID", userID, "lastTag", lastTag).Error("No more tags available for creating network")
		return nil, errors.New("no more tags available for creating network")
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get user by ID")
		return nil, err
	}

	sha256NetName := sha256.Sum256(fmt.Appendf([]byte{}, "%s-%s-%d", user.Username, cNetwork.SDNZone, newTag))
	netName := hex.EncodeToString(sha256NetName[:])

	net, err := db.CreateNetForUser(userID, netName, name, cNetwork.SDNZone, newTag, false, string(VNetStatusPreCreating))
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
