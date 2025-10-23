package main

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"slices"
	"sort"
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
		// check peers
		err := checkPeers(logger)
		if err != nil {
			logger.Error("Failed to check peers", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// check firewall
		err = checkFiewall(logger, fwConfig)
		if err != nil {
			logger.With("error", err).Error("Failed to check firewall")
			time.Sleep(10 * time.Second)
			continue
		}

		nets, err := internal.FetchNets(serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.Error("Failed to fetch nets status from main server", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		users, err := internal.FetchUsers(serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.Error("Failed to fetch users from main server", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		err = createPeers(logger, users)
		if err != nil {
			logger.Error("Failed to create VNets", "error", err)
		}

		err = enableNets(logger, nets, fwConfig)
		if err != nil {
			logger.Error("Failed to enable VNets", "error", err)
		}

		err = disableNets(logger, nets, fwConfig)
		if err != nil {
			logger.Error("Failed to delete VNets", "error", err)
		}

		err = updateNetsOnServer(logger, serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.Error("Failed to update nets on main server", "error", err)
		}

		time.Sleep(5 * time.Second)
	}
}

func disableNets(logger *slog.Logger, nets []internal.Net, fwConfig config.Firewall) error {
	localNets, err := db.GetAllSubnets()
	if err != nil {
		logger.Error("Failed to get all subnets from DB", "error", err)
		return err
	}

	for _, ln := range localNets {
		f := func(n internal.Net) bool { return n.Subnet == ln.Subnet }
		if slices.IndexFunc(nets, f) != -1 {
			continue
		}

		logger.Info("Deleting net", "subnet", ln.Subnet)

		for _, sp := range ln.Peers {
			peer, err := db.GetPeerByID(sp.ID)
			if err != nil {
				logger.Error("Failed to get peer from DB while disabling nets", "error", err, "peer_id", sp.ID)
				continue
			}
			err = shorewall.RemoveRule(util.CreateRule(fwConfig, "ACCEPT", peer.Address, ln.Subnet))
			if err != nil && !errors.Is(err, shorewall.ErrRuleNotFound) {
				logger.Error("Failed to delete firewall rule", "error", err)
				continue
			}
		}

		if err = shorewall.Reload(); err != nil {
			logger.Error("Failed to reload firewall", "error", err)
			continue
		}

		err = db.RemoveSubnet(ln.Subnet)
		if err != nil {
			logger.Error("Failed to remove subnet from DB", "error", err)
			continue
		}

		logger.Info("Successfully removed subnet", "subnet", ln.Subnet)
	}
	return nil
}

func createPeers(logger *slog.Logger, users []internal.User) error {
	for _, u := range users {
		peers, err := db.GetPeersByUserID(u.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Failed to get peer from DB for creation", "error", err, "user_id", u.ID)
			continue
		}
		// TODO: Eliminate redundant peers if len(peers) > u.NumberOFVPNConfigs
		if len(peers) >= int(u.NumberOFVPNConfigs) {
			continue
		}

		for i := range int(u.NumberOFVPNConfigs) - len(peers) {
			logger.Info("Creating new peer", "i", i, "user_id", u.ID)
			newAddr, err := util.NextAvailableAddress()
			if err != nil {
				logger.Error("Failed to generate new address", "error", err)
				continue
			}

			wgPeer, err := wg.NewWGConfig(newAddr)
			if err != nil {
				logger.Error("Failed to generate WireGuard config", "error", err)
				continue
			}

			err = db.NewPeer(wgPeer.PrivateKey, wgPeer.PublicKey, newAddr, u.ID)
			if err != nil {
				logger.Error("Failed to save peer to database", "error", err)
				continue
			}

			logger.Info("Successfully created new peer", "user_id", u.ID, "address", newAddr)
		}
	}

	return nil
}

func checkPeers(logger *slog.Logger) error {
	dbPeers, err := db.GetAllPeers()
	if err != nil {
		logger.Error("Failed to get peers from database", "error", err)
		return err
	}

	wgPeers, err := wg.ParsePeers()
	if err != nil {
		logger.Error("Failed to parse WireGuard peers", "error", err)
		return err
	}

	for _, peer := range dbPeers {
		dbp := wg.PeerFromDB(&peer)
		wgp, ok := wgPeers[dbp.PublicKey]
		if !ok {
			// not present, recreate it
			logger.Info("Peer not found in WireGuard config", "public_key", dbp.PublicKey)
			err = wg.CreatePeer(&dbp)
			if err != nil {
				logger.Error("Failed to create peer", "error", err)
				return err
			}
		} else {
			// is present, check if it's up to date
			if wgp.Address != dbp.Address {
				logger.Info("Peer address mismatch", "db_address", dbp.Address, "wg_address", wgp.Address)
				// recreate it with the correct fields
				err := wg.UpdatePeer(&dbp)
				if err != nil {
					logger.Error("Failed to update peer", "error", err)
					return err
				}
			}

			// delete present peers from the map, so that only peers not present in the database are left
			delete(wgPeers, dbp.PublicKey)
		}
	}

	// delete peers in wireguard that are not present in the database
	for _, peer := range wgPeers {
		err := wg.DeletePeer(&peer)
		if err != nil {
			logger.Error("Failed to delete peer", "error", err)
			return err
		}
	}
	return nil
}

func enableNets(logger *slog.Logger, nets []internal.Net, fwConfig config.Firewall) error {
	for _, n := range nets {
		if n.Subnet == "" {
			// When just created, the net has no subnet assigned yet
			continue
		}

		logger.Debug("Enabling net", "net", n.Subnet)

		var reloadShorewall bool = false
		for _, userID := range n.UserIDs {
			peers, err := db.GetPeersByUserID(userID)
			if err != nil {
				logger.Error("Failed to get peer from DB for enabling nets", "error", err, "user_id", userID)
				continue
			}
			for _, p := range peers {
				err = shorewall.AddRule(util.CreateRule(fwConfig, "ACCEPT", p.Address, n.Subnet))
				if err != nil && !errors.Is(err, shorewall.ErrRuleAlreadyExists) {
					logger.Error("Failed to add firewall rule", "error", err)
					continue
				} else if err == nil {
					reloadShorewall = true
				}

				err = db.NewSubnet(n.Subnet, p.ID)
				if err != nil {
					logger.Error("Failed to save subnet to database", "error", err)
					continue
				}
			}
		}

		if reloadShorewall {
			if err := shorewall.Reload(); err != nil {
				logger.Error("Failed to reload firewall", "error", err)
				continue
			}
			logger.Info("Successfully enabled net", "net", n)
		}
	}

	return nil
}

func updateNetsOnServer(logger *slog.Logger, endpoint, secret string) error {
	vpns, err := internal.FetchVPNConfigs(endpoint, secret)
	if err != nil {
		logger.Error("Failed to fetch VPN configs from main server", "error", err)
		return err
	}

	peers, err := db.GetAllPeers()
	if err != nil {
		logger.Error("Failed to get all peers from DB", "error", err)
		return err
	}

	// Creates VPN configs on the main server for local peers that don't have one yet
	for _, p := range peers {
		f := func(v internal.VPNUpdate) bool { return v.VPNIP == p.Address }
		if slices.IndexFunc(vpns, f) != -1 {
			continue
		}

		wgIface := wg.PeerFromDB(&p)
		base64Conf := base64.StdEncoding.EncodeToString([]byte(wgIface.String()))

		logger.Info("Creating VPN config on main server", "peer_id", p.ID)
		err = internal.CreateVPNConfig(endpoint, secret, internal.VPNCreate{
			VPNUpdate: internal.VPNUpdate{
				VPNConfig: base64Conf,
				VPNIP:     wgIface.Address,
			},
			UserID: p.UserID,
		})
		if err != nil {
			logger.Error("Failed to create VPN config on main server", "error", err)
			continue
		}
	}

	// Updates already existing VPN configs on the main server if they differ
	// from the local ones
	for _, v := range vpns {
		peer, err := db.GetPeerByAddress(v.VPNIP)
		if err != nil {
			logger.Error("Failed to get Peer from DB", "error", err)
			continue
		}

		wgIface := wg.PeerFromDB(peer)
		base64Conf := base64.StdEncoding.EncodeToString([]byte(wgIface.String()))
		if base64Conf == v.VPNConfig && wgIface.Address == v.VPNIP {
			continue
		}

		logger.Info("Updating VPN config on main server", "id", v.ID)
		err = internal.UpdateVPNConfig(endpoint, secret, internal.VPNUpdate{
			ID:        v.ID,
			VPNConfig: base64Conf,
			VPNIP:     wgIface.Address,
		})
	}

	return nil
}

func checkFiewall(logger *slog.Logger, fwConfig config.Firewall) error {
	// for all subnets in the db, check if there is a rule in shorewall
	subnets, err := db.GetAllSubnets()
	if err != nil {
		logger.With("error", err).Error("Failed to get all subnets from DB")
		return err
	}

	fwRules, err := shorewall.GetRules()
	if err != nil {
		logger.With("error", err).Error("Failed to get firewall rules")
		return err
	}

	// sort rules by Source
	sort.Slice(fwRules, func(i, j int) bool {
		if fwRules[i].Action != fwRules[j].Action {
			return fwRules[i].Action < fwRules[j].Action
		}
		if fwRules[i].Source != fwRules[j].Source {
			return fwRules[i].Source < fwRules[j].Source
		}
		return fwRules[i].Destination < fwRules[j].Destination
	})

	reloadFirewall := false

	for _, s := range subnets {
		for _, sp := range s.Peers {
			peer, err := db.GetPeerByID(sp.ID)
			if err != nil {
				logger.With("error", err).Error("Failed to get peer from DB")
				continue
			}

			rule := util.CreateRule(fwConfig, "ACCEPT", peer.Address, s.Subnet)

			// check if the rule exists in fwRules
			// using binary search since fwRules is sorted by Source
			index := sort.Search(len(fwRules), func(i int) bool {
				if fwRules[i].Action != rule.Action {
					return fwRules[i].Action > rule.Action
				}
				if fwRules[i].Source != rule.Source {
					return fwRules[i].Source > rule.Source
				}
				return fwRules[i].Destination >= rule.Destination
			})

			exists := index < len(fwRules) &&
				fwRules[index].Action == rule.Action &&
				fwRules[index].Source == rule.Source &&
				fwRules[index].Destination == rule.Destination
			if !exists {
				logger.Info("Firewall rule missing, adding it", "rule", rule)
				err = shorewall.AddRule(rule)
				if err != nil {
					logger.With("error", err).Error("Failed to add firewall rule")
					continue
				}
				reloadFirewall = true
			}
		}
	}

	// reload shorewall to apply changes
	if reloadFirewall {
		err = shorewall.Reload()
		if err != nil {
			logger.With("error", err).Error("Failed to reload firewall")
			return err
		}
	}

	return nil
}
