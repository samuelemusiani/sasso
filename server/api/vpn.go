package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/server/db"
)

func updateVPNConfig(w http.ResponseWriter, r *http.Request) {
	var vpnUpdate internal.VPNUpdate
	if err := json.NewDecoder(r.Body).Decode(&vpnUpdate); err != nil {
		logger.With("error", err).Error("Failed to decode VPN update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if vpnUpdate.UserID == 0 || vpnUpdate.VPNConfig == "" {
		logger.Error("UserID and VPNConfig are required")
		http.Error(w, "UserID and VPNConfig are required", http.StatusBadRequest)
		return
	}

	err := db.UpdateVPNConfig(vpnUpdate.VPNConfig, vpnUpdate.UserID)
	if err != nil {
		logger.With("error", err).Error("Failed to update VPN config in DB")
		http.Error(w, "Failed to update VPN config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getVPNConfigs(w http.ResponseWriter, r *http.Request) {
	vpnConfigs, err := db.GetAllVPNConfigs()
	if err != nil {
		logger.With("error", err).Error("Failed to get VPN configs from DB")
		http.Error(w, "Failed to get VPN configs", http.StatusInternalServerError)
		return
	}

	var vpns []internal.VPNUpdate
	for i := range vpnConfigs {
		vpns = append(vpns, internal.VPNUpdate{
			UserID:    vpnConfigs[i].ID,
			VPNConfig: *vpnConfigs[i].VPNConfig,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vpns); err != nil {
		logger.With("error", err).Error("Failed to encode VPN configs")
		http.Error(w, "Failed to encode VPN configs", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getUserVPNConfig(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	user, err := db.GetUserByID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get user from DB")
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	if user.VPNConfig == nil {
		http.Error(w, "No VPN config found for user", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "plain/text")

	config, err := base64.StdEncoding.DecodeString(*user.VPNConfig)

	w.Write([]byte(config))
}
