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

func internalUpdateVPNConfig(w http.ResponseWriter, r *http.Request) {
	var vpnUpdate internal.VPNProfile
	if err := json.NewDecoder(r.Body).Decode(&vpnUpdate); err != nil {
		logger.Error("Failed to decode VPN update request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	if vpnUpdate.ID == 0 || vpnUpdate.VPNConfig == "" {
		logger.Error("ID, UserID and VPNConfig are required")
		http.Error(w, "ID, UserID and VPNConfig are required", http.StatusBadRequest)

		return
	}

	err := db.UpdateVPNConfigByID(vpnUpdate.ID, vpnUpdate.VPNConfig, vpnUpdate.VPNIP)
	if err != nil {
		logger.Error("Failed to update VPN config in DB", "error", err)
		http.Error(w, "Failed to update VPN config", http.StatusInternalServerError)

		return
	}
}

func internalGetVPNConfigs(w http.ResponseWriter, r *http.Request) {
	vpnConfigs, err := db.GetAllVPNConfigs()
	if err != nil {
		logger.Error("Failed to get VPN configs from DB", "error", err)
		http.Error(w, "Failed to get VPN configs", http.StatusInternalServerError)

		return
	}

	var vpns []internal.VPNProfile
	for i := range vpnConfigs {
		vpns = append(vpns, internal.VPNProfile{
			ID:        vpnConfigs[i].ID,
			VPNConfig: vpnConfigs[i].VPNConfig,
			VPNIP:     vpnConfigs[i].VPNIP,
			UserID:    vpnConfigs[i].UserID,
		})
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(vpns); err != nil {
		logger.Error("Failed to encode VPN configs", "error", err)
		http.Error(w, "Failed to encode VPN configs", http.StatusInternalServerError)

		return
	}
}

type returnConfig struct {
	ID        uint   `json:"id"`
	VPNConfig string `json:"vpn_config"`
}

func getUserVPNConfigs(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	vpnConfigs, err := db.GetVPNConfigsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get VPN configs from DB", "error", err)
		http.Error(w, "Failed to get VPN configs", http.StatusInternalServerError)

		return
	}

	if len(vpnConfigs) == 0 {
		http.Error(w, "No VPN config found for user", http.StatusNotFound)
		return
	}

	returnConfigs := make([]returnConfig, 0, len(vpnConfigs))
	for i := range vpnConfigs {
		// Just to check if the config is valid base64
		if vpnConfigs[i].VPNConfig == "" {
			// New generated configs are empty, skip validation and do not return them
			continue
		}

		_, err := base64.StdEncoding.DecodeString(vpnConfigs[i].VPNConfig)
		if err != nil {
			logger.Error("Failed to decode VPN config", "error", err)
			http.Error(w, "Failed to decode VPN config", http.StatusInternalServerError)

			return
		}

		returnConfigs = append(returnConfigs, returnConfig{
			ID:        vpnConfigs[i].ID,
			VPNConfig: vpnConfigs[i].VPNConfig,
		})
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(returnConfigs); err != nil {
		logger.Error("Failed to encode VPN configs", "error", err)
		http.Error(w, "Failed to encode VPN configs", http.StatusInternalServerError)

		return
	}
}

func addVPNConfig(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	nConfigs, err := db.CountVPNConfigsByUserID(userID)
	if err != nil {
		logger.Error("failed to get user VPN config count", "error", err)
		http.Error(w, "Failed to get VPN config count", http.StatusInternalServerError)

		return
	}

	if nConfigs >= int64(vpnConfigs.MaxProfilesPerUser) {
		http.Error(w, "VPN config limit reached", http.StatusBadRequest)
		return
	}

	// To add a VPN config with put an empty config, the actual config will be
	// updated later by the internalUpdateVPNConfig endpoint (aka from the VPN
	// service worker)

	err = db.CreateVPNConfig("", "", userID)
	if err != nil {
		logger.Error("failed to create VPN config", "error", err)
		http.Error(w, "Failed to create VPN config", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteVPNConfig(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	svpnID := chi.URLParam(r, "id")

	vpnID, err := strconv.ParseUint(svpnID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid VPN config ID format", http.StatusBadRequest)
		return
	}

	vpnConfig, err := db.GetVPNConfigByID(uint(vpnID))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			http.Error(w, "VPN config not found", http.StatusNotFound)
			return
		}

		logger.Error("Failed to get VPN config by ID", "error", err)
		http.Error(w, "Failed to get VPN config", http.StatusInternalServerError)

		return
	}

	if vpnConfig.UserID != userID {
		http.Error(w, "VPN config not found", http.StatusNotFound)
		return
	}

	err = db.DeleteVPNConfigByID(uint(vpnID))
	if err != nil {
		logger.Error("Failed to delete VPN config", "error", err)
		http.Error(w, "Failed to delete VPN config", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
