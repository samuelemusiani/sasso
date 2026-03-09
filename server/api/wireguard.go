package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/server/db"
)

// convert from internal WireguardPeer to db WireguardPeer
func internalWGPeerToDBWGPeer(p *internal.WireguardPeer) *db.WireguardPeer {
	allowedIPs := make([]db.WireguardAllowedIP, len(p.AllowedIPs))
	for i, ip := range p.AllowedIPs {
		allowedIPs[i] = db.WireguardAllowedIP{
			IP: ip,
		}
	}

	return &db.WireguardPeer{
		ID:              p.ID,
		IP:              p.IP,
		PeerPrivateKey:  p.PeerPrivateKey,
		ServerPublicKey: p.ServerPublicKey,
		Endpoint:        p.Endpoint,
		AllowedIPs:      allowedIPs,
		UserID:          p.UserID,
	}
}

// convert from db WireguardPeer to internal WireguardPeer
func dbWGPeerToInternalWGPeer(p *db.WireguardPeer) *internal.WireguardPeer {
	allowedIPs := make([]string, len(p.AllowedIPs))
	for i, ip := range p.AllowedIPs {
		allowedIPs[i] = ip.IP
	}

	return &internal.WireguardPeer{
		ID:              p.ID,
		IP:              p.IP,
		PeerPrivateKey:  p.PeerPrivateKey,
		ServerPublicKey: p.ServerPublicKey,
		Endpoint:        p.Endpoint,
		AllowedIPs:      allowedIPs,
		UserID:          p.UserID,
	}
}

func internalUpdateWireguardPeer(w http.ResponseWriter, r *http.Request) {
	var vpnUpdate internal.WireguardPeer
	if err := json.NewDecoder(r.Body).Decode(&vpnUpdate); err != nil {
		logger.Error("Failed to decode wireguard peer update request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	err := db.UpdateWireguardPeer(internalWGPeerToDBWGPeer(&vpnUpdate))
	if err != nil {
		logger.Error("Failed to update wireguard peer in DB", "error", err)
		http.Error(w, "Failed to update VPN config", http.StatusInternalServerError)

		return
	}
}

func internalWireguardPeers(w http.ResponseWriter, _ *http.Request) {
	vpnConfigs, err := db.GetAllWireguardPeers()
	if err != nil {
		logger.Error("Failed to get wireguard peers from DB", "error", err)
		http.Error(w, "Failed to get wireguard peers", http.StatusInternalServerError)

		return
	}

	vpns := make([]internal.WireguardPeer, 0, len(vpnConfigs))
	for i := range vpnConfigs {
		vpns = append(vpns, *dbWGPeerToInternalWGPeer(&vpnConfigs[i]))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(vpns); err != nil {
		logger.Error("Failed to encode wireguard peers", "error", err)
		http.Error(w, "Failed to encode wireguard peers", http.StatusInternalServerError)

		return
	}
}

type returnConfig struct {
	ID        uint   `json:"id"`
	VPNConfig string `json:"vpn_config"`
}

func getUserWireguardPeers(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	wgPeers, err := db.GetWireguardPeerByUserID(userID)
	if err != nil {
		logger.Error("Failed to get wireguard peers for user from DB", "user_id", userID, "error", err)
		http.Error(w, "Failed to get wireguard peers for user", http.StatusInternalServerError)

		return
	}

	if len(wgPeers) == 0 {
		http.Error(w, "No wireguard peers found for user", http.StatusNotFound)

		return
	}

	returnConfigs := make([]returnConfig, 0, len(wgPeers))
	for i := range wgPeers {
		// New generated configs are empty, skip validation and do not return them
		if wgPeers[i].IP == "" {
			continue
		}

		wgPeerString := dbWGPeerToInternalWGPeer(&wgPeers[i]).String()
		returnConfigs = append(returnConfigs, returnConfig{
			ID:        wgPeers[i].ID,
			VPNConfig: base64.StdEncoding.EncodeToString([]byte(wgPeerString)),
		})
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(returnConfigs); err != nil {
		logger.Error("Failed to encode wireguard peers for user", "user_id", userID, "error", err)
		http.Error(w, "Failed to encode wireguard peers for user", http.StatusInternalServerError)

		return
	}
}

func addWireguardPeer(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	nConfigs, err := db.CountWireguardPeersByUserID(userID)
	if err != nil {
		logger.Error("failed to count wireguard peers for user", "user_id", userID, "error", err)
		http.Error(w, "Failed to count wireguard peers for user", http.StatusInternalServerError)

		return
	}

	if nConfigs >= int64(vpnConfigs.MaxWireguardProfilesPerUser) {
		http.Error(w, "Maximum number of wireguard peers reached for user", http.StatusBadRequest)

		return
	}

	// To add a VPN config, we put an empty config in the DB. The actual config
	// will be updated later by the internalUpdateVPNConfig endpoint (aka from
	// the VPN service worker)

	err = db.CreateWireguardPeer(userID)
	if err != nil {
		logger.Error("failed to create wireguard peer for user", "user_id", userID, "error", err)
		http.Error(w, "Failed to create wireguard peer for user", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteWireguardPeer(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	svpnID := chi.URLParam(r, "id")

	vpnID, err := strconv.ParseUint(svpnID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid VPN config ID format", http.StatusBadRequest)

		return
	}

	err = db.DeleteWireguardPeerByIDAndUserID(uint(vpnID), userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			http.Error(w, "Wireguard peer not found", http.StatusNotFound)

			return
		}

		logger.Error("Failed to delete wireguard peer for user from DB", "user_id", userID, "vpn_id", vpnID, "error", err)
		http.Error(w, "Failed to delete wireguard peer for user", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
