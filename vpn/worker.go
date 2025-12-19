package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"slices"
	"sort"
	"time"

	shorewall "github.com/samuelemusiani/go-shorewall"
	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/util"
	"samuelemusiani/sasso/vpn/wg"
)

func checkConfig(serverConfig config.Server, fwConfig config.Firewall, vpnSubnet string) error {
	if serverConfig.Endpoint == "" {
		return errors.New("server endpoint is empty")
	}

	if serverConfig.Secret == "" {
		return errors.New("server secret is empty")
	}

	// Endpoint should be a valid URL
	_, err := url.Parse(serverConfig.Endpoint)
	if err != nil {
		return errors.New("server endpoint is not a valid URL")
	}

	if fwConfig.VPNZone == "" {
		return errors.New("firewall VPN zone is empty")
	}

	if fwConfig.SassoZone == "" {
		return errors.New("firewall Sasso zone is empty")
	}

	if vpnSubnet == "" {
		return errors.New("VPN subnet is empty")
	}

	if _, _, err := net.ParseCIDR(vpnSubnet); err != nil {
		return fmt.Errorf("VPN subnet %s is not a valid CIDR: %w", vpnSubnet, err)
	}

	return nil
}

func checkFirewallStatus(fwConfig config.Firewall) error {
	v, err := shorewall.GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get shorewall version: %w", err)
	}

	slog.Info("Shorewall version", "version", v)

	zones, err := shorewall.GetZones()
	if err != nil {
		return fmt.Errorf("failed to get shorewall zones: %w", err)
	}

	fwZones := []string{fwConfig.VPNZone, fwConfig.SassoZone}
	for _, z := range fwZones {
		if !slices.ContainsFunc(zones, func(sz shorewall.Zone) bool {
			return sz.Name == z
		}) {
			return fmt.Errorf("shorewall zone %s not found", z)
		}
	}

	return nil
}

func worker(logger *slog.Logger, serverConfig config.Server, fwConfig config.Firewall, vpnSubnet string) {
	logger.Info("Worker started")

	for {
		err := checkPeers(logger)
		if err != nil {
			logger.Error("Failed to check peers", "error", err)
			time.Sleep(10 * time.Second)

			continue
		}

		err = checkFirewall(logger, fwConfig)
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

		vpnConfigs, err := internal.FetchVPNConfigs(serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.Error("Failed to fetch VPN configs from main server", "error", err)
			time.Sleep(10 * time.Second)

			continue
		}

		err = deletePeers(logger, vpnConfigs, fwConfig)
		if err != nil {
			logger.Error("Failed to delete peers", "error", err)
		}

		err = createPeers(logger, vpnConfigs, vpnSubnet)
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

		err = updateNetsOnServer(logger, vpnConfigs, serverConfig.Endpoint, serverConfig.Secret)
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

	// Remove firewall rules and DB entries for nets that are no longer present
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

func deletePeers(logger *slog.Logger, vpnConfigs []internal.VPNProfile, fwConfig config.Firewall) error {
	// We could optimize this by looking at which users have reduced their number
	// of VPN configs, but for now we only iterate over all peers and check if they
	// are still needed.
	peers, err := db.GetAllPeers()
	if err != nil {
		logger.Error("Failed to get all peers from DB", "error", err)
		return err
	}

	for _, p := range peers {
		if !slices.ContainsFunc(vpnConfigs, func(v internal.VPNProfile) bool {
			return v.VPNIP == p.Address
		}) {
			wgPeer := wg.PeerFromDB(&p)

			err = wg.DeletePeer(&wgPeer)
			if err != nil {
				logger.Error("Failed to delete peer from WireGuard", "error", err, "peer_id", p.ID)
				continue
			}

			nets, err := db.GetSubnetsByPeerID(p.ID)
			if err != nil {
				logger.Error("Failed to get subnets for deleted peer", "error", err, "peer_id", p.ID)
				continue
			}

			for _, n := range nets {
				logger.Warn("Deleting firewall rule for deleted peer", "peer_id", p.ID, "subnet", n.Subnet)

				err = shorewall.RemoveRule(util.CreateRule(fwConfig, "ACCEPT", p.Address, n.Subnet))
				if err != nil && !errors.Is(err, shorewall.ErrRuleNotFound) {
					logger.Error("Failed to delete firewall rule for deleted peer", "error", err, "peer_id", p.ID, "subnet", n.Subnet)
					continue
				} else {
					logger.Info("Successfully deleted firewall rule for deleted peer", "peer_id", p.ID, "subnet", n.Subnet)
				}
			}

			if err = shorewall.Reload(); err != nil {
				logger.Error("Failed to reload firewall after deleting peer", "error", err, "peer_id", p.ID)
				continue
			}

			err = db.DeletePeerByID(p.ID)
			if err != nil {
				logger.Error("Failed to delete peer from DB", "error", err, "peer_id", p.ID)
				continue
			}

			logger.Info("Successfully deleted peer", "peer_id", p.ID, "address", p.Address)
		}
	}

	return nil
}

func createPeers(logger *slog.Logger, vpnConfigs []internal.VPNProfile, vpnSubnet string) error {
	peers, err := db.GetAllPeers()
	if err != nil {
		logger.Error("Failed to get all peers from DB", "error", err)
		return err
	}

	for _, v := range vpnConfigs {
		if slices.ContainsFunc(peers, func(p db.Peer) bool {
			return p.ID == v.ID
		}) {
			// Peer already exists
			continue
		}

		// Create new peer
		logger.Info("Creating new peer", "ID", v.ID, "user_id", v.UserID)

		newAddr, err := util.NextAvailableAddress(vpnSubnet)
		if err != nil {
			logger.Error("Failed to generate new address", "error", err)
			continue
		}

		wgPeer, err := wg.NewWGConfig(newAddr)
		if err != nil {
			logger.Error("Failed to generate WireGuard config", "error", err)
			continue
		}

		err = db.NewPeer(wgPeer.PrivateKey, wgPeer.PublicKey, newAddr, v.UserID)
		if err != nil {
			logger.Error("Failed to save peer to database", "error", err)
			continue
		}

		logger.Info("Successfully created new peer", "user_id", v.UserID, "address", newAddr)
	}

	return nil
}

// checkPeers compares the peers in the database with the peers in the WireGuard
// and makes sure they are in sync. The DB state is considered the source of
// truth.
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

		reloadShorewall := false

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
				if err != nil && !errors.Is(err, db.ErrAlreadyExists) {
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

func updateNetsOnServer(logger *slog.Logger, vpns []internal.VPNProfile, endpoint, secret string) error {
	// Updates existing VPN configs on the main server if they differ from the
	// local ones
	peers, err := db.GetAllPeers()
	if err != nil {
		logger.Error("Failed to get all peers from DB", "error", err)
		return err
	}

	for _, v := range vpns {
		idx := slices.IndexFunc(peers, func(p db.Peer) bool {
			return p.ID == v.ID
		})

		if idx == -1 {
			logger.Warn("Peer not found in DB, skipping update. This should not happen.", "id", v.ID)
			continue
		}

		peer := &peers[idx]

		wgIface := wg.PeerFromDB(peer)

		base64Conf := base64.StdEncoding.EncodeToString([]byte(wgIface.String()))
		if base64Conf == v.VPNConfig && wgIface.Address == v.VPNIP {
			continue
		}

		logger.Info("Updating VPN config on main server", "id", v.ID)

		err = internal.UpdateVPNConfig(endpoint, secret, internal.VPNProfile{
			ID:        v.ID,
			VPNConfig: base64Conf,
			VPNIP:     wgIface.Address,
		})
		if err != nil {
			logger.Error("Failed to update VPN config on main server", "error", err, "id", v.ID)
			continue
		}
	}

	return nil
}

// checkFirewall checks that all firewall rules for the peers in the database
// are actually present in shorewall, and adds them if they are missing.
func checkFirewall(logger *slog.Logger, fwConfig config.Firewall) error {
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
