package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/util"
	"samuelemusiani/sasso/vpn/wg"

	shorewall "github.com/samuelemusiani/go-shorewall"
	"gorm.io/gorm"
)

func worker(logger *slog.Logger, serverConfig config.Server, fwConfig config.Firewall) {
	logger.Info("Worker started")

	for {
		nets, err := internal.FetchNets(serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.With("error", err).Error("Failed to fetch nets status from main server")
			time.Sleep(10 * time.Second)
			continue
		}

		users, err := internal.FetchUsers(serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.With("error", err).Error("Failed to fetch users from main server")
			time.Sleep(10 * time.Second)
			continue
		}

		err = createPeers(logger, users)
		if err != nil {
			logger.With("error", err).Error("Failed to create VNets")
		}

		err = enableNets(logger, nets, fwConfig)
		if err != nil {
			logger.With("error", err).Error("Failed to enable VNets")
		}

		err = disableNets(logger, nets, fwConfig)
		if err != nil {
			logger.With("error", err).Error("Failed to delete VNets")
		}

		err = updateNetsOnServer(logger, serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.With("error", err).Error("Failed to update nets on main server")
		}

		time.Sleep(5 * time.Second)
	}
}

func disableNets(logger *slog.Logger, nets []internal.Net, fwConfig config.Firewall) error {
	localNets, err := db.GetAllSubnets()
	if err != nil {
		logger.With("error", err).Error("Failed to get all subnets from DB")
		return err
	}

	for _, ln := range localNets {
		f := func(n internal.Net) bool { return n.Subnet == ln.Subnet }
		if slices.IndexFunc(nets, f) != -1 {
			continue
		}

		logger.Info("Deleting net", "subnet", ln.Subnet)

		iface, err := db.GetPeerByID(ln.PeerID)
		if err != nil {
			logger.With("error", err).Error("Failed to get peer from DB")
			continue
		}
		err = shorewall.RemoveRule(shorewall.Rule{
			Action:      "ACCEPT",
			Source:      fmt.Sprintf("%s:%s", fwConfig.VPNZone, iface.Address),
			Destination: fmt.Sprintf("%s:%s", fwConfig.SassoZone, ln.Subnet),
		})
		if err != nil && !errors.Is(err, shorewall.ErrRuleNotFound) {
			logger.With("error", err).Error("Failed to delete firewall rule")
			continue
		}

		if err = shorewall.Reload(); err != nil {
			logger.With("error", err).Error("Failed to reload firewall")
			continue
		}

		err = db.RemoveSubnet(ln.Subnet)
		if err != nil {
			logger.With("error", err).Error("Failed to remove subnet from DB")
			continue
		}

		logger.Info("Successfully removed subnet", "subnet", ln.Subnet)
	}
	return nil
}

func createPeers(logger *slog.Logger, users []internal.User) error {
	for _, u := range users {
		_, err := db.GetPeerByUserID(u.ID)

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.With("error", err).Error("Failed to get peer from DB")
			continue
		} else if err == nil {
			// peer already exists, skip
			continue
		}
		logger.Info("Creating new peer", "user_id", u.ID)

		newAddr, err := util.NextAvailableAddress()
		if err != nil {
			logger.With("error", err).Error("Failed to generate new address")
			continue
		}

		wgPeer, err := wg.NewWGConfig(newAddr)
		if err != nil {
			logger.With("error", err).Error("Failed to generate WireGuard config")
			continue
		}

		err = db.NewPeer(wgPeer.PrivateKey, wgPeer.PublicKey, newAddr, u.ID)
		if err != nil {
			logger.With("error", err).Error("Failed to save peer to database")
			continue
		}

		logger.Info("Successfully created new peer", "user_id", u.ID, "address", newAddr)
	}

	return nil
}

func enableNets(logger *slog.Logger, nets []internal.Net, fwConfig config.Firewall) error {
	for _, n := range nets {
		if n.Subnet == "" {
			// When just created, the net has no subnet assigned yet
			continue
		}

		// Check if the subnet associated to the net already exists in the DB
		exist, err := db.CheckSubnetExists(n.Subnet)
		if err != nil {
			logger.With("error", err).Error("Failed to check if subnet exists")
			continue
		}

		// If it exists, skip it
		if exist {
			continue
		}

		logger.Debug("Enabling net", "net", n.Subnet)

		iface, err := db.GetPeerByUserID(n.UserID)
		if err != nil {
			logger.With("error", err).Error("Failed to get peer from DB")
			continue
		}

		err = shorewall.AddRule(shorewall.Rule{
			Action:      "ACCEPT",
			Source:      fmt.Sprintf("%s:%s", fwConfig.VPNZone, iface.Address),
			Destination: fmt.Sprintf("%s:%s", fwConfig.SassoZone, n.Subnet),
		})

		if err != nil && !errors.Is(err, shorewall.ErrRuleAlreadyExists) {
			logger.With("error", err).Error("Failed to add firewall rule")
			continue
		}

		if err = shorewall.Reload(); err != nil {
			logger.With("error", err).Error("Failed to reload firewall")
			continue
		}

		wgIface := wg.PeerFromDB(iface)
		err = wg.CreatePeer(&wgIface)
		if err != nil {
			logger.With("error", err).Error("Failed to create WireGuard peer")
			continue
		}

		err = db.NewSubnet(n.Subnet, iface.ID)
		if err != nil {
			logger.With("error", err).Error("Failed to save subnet to database")
			continue
		}
		logger.Info("Successfully enabled net", "net", n)
	}

	return nil
}

func updateNetsOnServer(logger *slog.Logger, endpoint, secret string) error {
	vpns, err := internal.FetchVPNConfigs(endpoint, secret)
	if err != nil {
		logger.With("error", err).Error("Failed to fetch VPN configs from main server")
		return err
	}

	localPeers, err := db.GetAllPeers()
	if err != nil {
		logger.With("error", err).Error("Failed to fetch local peers from DB")
		return err
	}

	for _, i := range localPeers {
		if slices.IndexFunc(vpns, func(v internal.VPNUpdate) bool { return v.UserID == i.UserID }) == -1 {
			vpns = append(vpns, internal.VPNUpdate{
				UserID:    i.UserID,
				VPNConfig: "",
			})
		}
	}

	for _, v := range vpns {
		iface, err := db.GetPeerByUserID(v.UserID)
		if err != nil {
			logger.With("error", err).Error("Failed to get Peer from DB")
			continue
		}

		wgIface := wg.PeerFromDB(iface)
		base64Conf := base64.StdEncoding.EncodeToString([]byte(wgIface.String()))
		if base64Conf == v.VPNConfig {
			continue
		}

		logger.With("user_id", v.UserID).Info("Updating VPN config on main server")
		err = internal.UpdateVPNConfig(endpoint, secret, internal.VPNUpdate{
			UserID:    v.UserID,
			VPNConfig: base64Conf,
		})
	}

	return nil
}
