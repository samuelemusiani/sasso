package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"time"

	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/fw"
	"samuelemusiani/sasso/vpn/util"
	"samuelemusiani/sasso/vpn/wg"
)

func checkConfig(serverConfig config.Server) error {
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

	return nil
}

func dbWireguardPeersToInternal(wgPeers []db.WireguardPeer) []internal.WireguardPeer {
	internalPeers := make([]internal.WireguardPeer, len(wgPeers))
	for i, p := range wgPeers {
		allowedIPs := make([]string, len(p.AllowedIPs))
		for j, ip := range p.AllowedIPs {
			allowedIPs[j] = ip.IP
		}

		internalPeers[i] = internal.WireguardPeer{
			ID:              p.ID,
			IP:              p.IP,
			PeerPrivateKey:  p.PeerPrivateKey,
			ServerPublicKey: p.ServerPublicKey,
			Endpoint:        p.Endpoint,
			AllowedIPs:      allowedIPs,
			UserID:          p.UserID,
		}
	}

	return internalPeers
}

func dbNetsToInternal(dbNets []db.Net) []internal.Net {
	internalNets := make([]internal.Net, len(dbNets))
	for i, n := range dbNets {
		userIDs := make([]uint, len(n.UserIDs))
		for j, userID := range n.UserIDs {
			userIDs[j] = userID.UserID
		}

		internalNets[i] = internal.Net{
			ID:        n.ID,
			Zone:      n.Zone,
			Name:      n.Name,
			Tag:       n.Tag,
			Subnet:    n.Subnet,
			Gateway:   n.Gateway,
			Broadcast: n.Broadcast,
			UserIDs:   userIDs,
		}
	}

	return internalNets
}

func worker(parentCtx context.Context, logger *slog.Logger, firewall fw.Firewall, serverConfig config.Server, vpnSubnet string) {
	logger.Info("worker started")

	var (
		timeToSleep time.Duration
		err         error
	)

	var (
		wireguardPeers []db.WireguardPeer
		nets           []db.Net

		internalWireguardPeers []internal.WireguardPeer
	)

	wireguardPeers, err = db.GetAllWireguardPeers()
	if err != nil {
		logger.Error("Failed to fetch WireGuard peers from DB", "error", err)

		goto start_loop
	}

	internalWireguardPeers = dbWireguardPeersToInternal(wireguardPeers)

	err = applyWireguardPeers(logger, internalWireguardPeers)
	if err != nil {
		logger.Error("Failed to apply WireGuard peers from DB", "error", err)
	}

	nets, err = db.GetAllNets()
	if err != nil {
		logger.Error("Failed to fetch nets from DB", "error", err)

		goto start_loop
	}

	err = applyNetsToFirewall(firewall, internalWireguardPeers, dbNetsToInternal(nets))
	if err != nil {
		logger.Error("Failed to apply nets from DB", "error", err)

		goto start_loop
	}

start_loop:
	for {
		if err != nil {
			timeToSleep = 10 * time.Second
		} else {
			timeToSleep = 5 * time.Second
		}

		select {
		case <-time.After(timeToSleep):
		case <-parentCtx.Done():
			logger.Info("worker shutting down")

			return
		}

		// This worker takes care of two things:
		// 1. Wireguard peers
		// 2. Firewall rules for the Wireguard peers
		//
		// We have 3 states for these resources:
		// 1. Server main state (what we want)
		// 2. Wireguard peers state (what we have)
		// 3. Wireguard peers DB (what we rember we had)
		//
		// Flow:
		// 1. Pull from Main server (if fail pass over)
		// 2. Update DB (if no update from main server use last stored state)
		// 3. Update Wireguard peers State (with Server state or last DB state)
		// 4. Repeat

		var wireguardPeers []internal.WireguardPeer

		wireguardPeers, err = internal.FetchWireguardPeers(parentCtx, serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.Error("failed to fetch wireguard peers from main server", "error", err)

			continue
		}

		oldWireguardPeers := make([]internal.WireguardPeer, len(wireguardPeers))
		copy(oldWireguardPeers, wireguardPeers)

		wireguardPeers, err = fillEmptyWireguardPeers(logger, wireguardPeers, vpnSubnet)
		if err != nil {
			logger.Error("failed to fill empty wireguard peers", "error", err)

			continue
		}

		err = updateDBWithServerWireguardPeers(wireguardPeers)
		if err != nil {
			logger.Error("failed to update DB with server wireguard peers", "error", err)

			continue
		}

		err = applyWireguardPeers(logger, wireguardPeers)
		if err != nil {
			logger.Error("failed to apply wireguard peers", "error", err)

			continue
		}

		err = pushWireguardPeersToServer(parentCtx, logger, serverConfig, oldWireguardPeers, wireguardPeers)
		if err != nil {
			logger.Error("Failed to push wireguard peers to main server", "error", err)

			continue
		}

		// ------- Nets -------

		nets, err := internal.FetchNets(parentCtx, serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.Error("Failed to fetch nets status from main server", "error", err)

			continue
		}

		err = updateDBWithNets(nets)
		if err != nil {
			logger.Error("Failed to update DB with nets from main server", "error", err)
		}

		err = applyNetsToFirewall(firewall, wireguardPeers, nets)
		if err != nil {
			logger.Error("Failed to apply nets", "error", err)
		}
	}
}

// To generate a new Wireguard peer, the server sends use an empty one with only
// ID and UserID fields filled. This function fills the other fields with new
// generated values.
func fillEmptyWireguardPeers(logger *slog.Logger, vpnConfigs []internal.WireguardPeer, vpnSubnet string) ([]internal.WireguardPeer, error) {
	var usedAddresses []string

	for i, v := range vpnConfigs {
		if v.IP != "" {
			continue
		}

		logger.Info("Creating new peer", "ID", v.ID, "user_id", v.UserID)

		newAddr, err := util.NextAvailableAddressWithAddresses(vpnSubnet, usedAddresses)
		if err != nil {
			return nil, fmt.Errorf("failed to generate new address: %w", err)
		}

		wgPeer, err := wg.NewPeer(newAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to generate WireGuard config: %w", err)
		}

		vpnConfigs[i].IP = newAddr
		vpnConfigs[i].ServerPublicKey = wgPeer.ServerPublicKey
		vpnConfigs[i].PeerPrivateKey = wgPeer.PeerPrivateKey
		vpnConfigs[i].Endpoint = wgPeer.Endpoint
		vpnConfigs[i].AllowedIPs = wgPeer.AllowedIPs

		logger.Debug("Generated new Wireguard peer", "peer", vpnConfigs[i])

		usedAddresses = append(usedAddresses, newAddr)
	}

	return vpnConfigs, nil
}

func updateDBWithServerWireguardPeers(vpnConfigs []internal.WireguardPeer) error {
	dbWireguardPeers := make([]db.WireguardPeer, 0, len(vpnConfigs))
	for _, c := range vpnConfigs {
		allowedIPs := make([]db.WireguardAllowedIP, len(c.AllowedIPs))
		for i, ip := range c.AllowedIPs {
			allowedIPs[i] = db.WireguardAllowedIP{
				IP: ip,
			}
		}

		dbWireguardPeers = append(dbWireguardPeers, db.WireguardPeer{
			IP:              c.IP,
			PeerPrivateKey:  c.PeerPrivateKey,
			ServerPublicKey: c.ServerPublicKey,
			Endpoint:        c.Endpoint,
			AllowedIPs:      allowedIPs,
			UserID:          c.UserID,
		})
	}

	err := db.UpdateAllWireguardPeers(dbWireguardPeers)
	if err != nil {
		return fmt.Errorf("failed to update DB with server WireGuard peers: %w", err)
	}

	return nil
}

// applyWireguardPeers applies the given WireGuard peers to the local WireGuard
// interface.
func applyWireguardPeers(logger *slog.Logger, wireguardPeers []internal.WireguardPeer) error {
	currentPeers, err := wg.ParsePeers()
	if err != nil {
		return fmt.Errorf("failed to parse current WireGuard peers: %w", err)
	}

	vpnConfigsMap := make(map[string]internal.WireguardPeer)
	for _, c := range wireguardPeers {
		vpnConfigsMap[c.ServerPublicKey] = c
	}

	// delete peers that are not present in wireguardPeers slice
	for publicKey, peer := range currentPeers {
		if _, ok := vpnConfigsMap[publicKey]; ok {
			continue
		}

		logger.Info("peer not found in server config, deleting it", "public_key", publicKey)

		err = wg.DeletePeer(&peer)
		if err != nil {
			return fmt.Errorf("failed to delete peer: %w", err)
		}

		logger.Info("successfully deleted peer", "public_key", publicKey)
	}

	// create new peers and update existing ones
	for _, c := range wireguardPeers {
		wgPeer := wg.Peer{
			IP:              c.IP,
			PeerPrivateKey:  c.PeerPrivateKey,
			ServerPublicKey: c.ServerPublicKey,
			Endpoint:        c.Endpoint,
			AllowedIPs:      c.AllowedIPs,
		}

		if peer, ok := currentPeers[c.ServerPublicKey]; !ok {
			logger.Info("peer not found in current WireGuard config, creating it", "public_key", c.ServerPublicKey)

			err = wg.CreatePeer(&wgPeer)
			if err != nil {
				return fmt.Errorf("failed to create peer: %w", err)
			}

			logger.Info("successfully created peer", "public_key", c.ServerPublicKey)
		} else if !peer.Equal(wgPeer) {
			logger.Info("peer found in current WireGuard config but differs from server config, updating it", "public_key", c.ServerPublicKey)

			err = wg.UpdatePeer(&wgPeer)
			if err != nil {
				return fmt.Errorf("failed to update peer: %w", err)
			}

			logger.Info("successfully updated peer", "public_key", c.ServerPublicKey)
		}
	}

	return nil
}

func pushWireguardPeersToServer(parentCtx context.Context, logger *slog.Logger, serverConfig config.Server, oldPeers, newPeers []internal.WireguardPeer) error {
	for _, newPeer := range newPeers {
		oldPeerIndex := slices.IndexFunc(oldPeers, func(p internal.WireguardPeer) bool {
			return p.ID == newPeer.ID
		})

		if oldPeerIndex == -1 {
			return fmt.Errorf("old peer not found for new peer with ID %d", newPeer.ID)
		}

		if oldPeers[oldPeerIndex].Equals(newPeer) {
			continue
		}

		err := internal.UpdateWireguardPeer(parentCtx, serverConfig.Endpoint, serverConfig.Secret, newPeer)
		if err != nil {
			return fmt.Errorf("failed to update WireGuard peer on main server: %w", err)
		}

		logger.Info("updated WireGuard peer on main server", "peer_id", newPeer.ID)
	}

	return nil
}

func updateDBWithNets(nets []internal.Net) error {
	dbNets := make([]db.Net, len(nets))
	for i, n := range nets {
		dbUserIDs := make([]db.NetUserID, len(n.UserIDs))
		for j, userID := range n.UserIDs {
			dbUserIDs[j] = db.NetUserID{
				UserID: userID,
			}
		}

		dbNets[i] = db.Net{
			Zone:      n.Zone,
			Name:      n.Name,
			Tag:       n.Tag,
			Subnet:    n.Subnet,
			Gateway:   n.Gateway,
			Broadcast: n.Broadcast,
			UserIDs:   dbUserIDs,
		}
	}

	err := db.UpdateAllNets(dbNets)
	if err != nil {
		return fmt.Errorf("failed to update DB with nets from main server: %w", err)
	}

	return nil
}

func applyNetsToFirewall(firewall fw.Firewall, wireguardPeers []internal.WireguardPeer, nets []internal.Net) error {
	var rules []fw.Rule

	for _, peer := range wireguardPeers {
		for _, net := range nets {
			if slices.Contains(net.UserIDs, peer.UserID) {
				rules = append(rules, firewall.CreateAllowRule(peer.IP, net.Subnet))
			}
		}
	}

	return firewall.ApplyRules(rules)
}
