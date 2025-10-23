package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/server/db"
)

func internalUpdateVPNConfig(w http.ResponseWriter, r *http.Request) {
	var vpnUpdate internal.VPNUpdate
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

func internalCreateVPNConfig(w http.ResponseWriter, r *http.Request) {
	var vpnCreate internal.VPNUpdate
	if err := json.NewDecoder(r.Body).Decode(&vpnCreate); err != nil {
		logger.Error("Failed to decode VPN create request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if vpnCreate.VPNConfig == "" || vpnCreate.VPNIP == "" {
		logger.Error("UserID, VPNConfig and VPNIP are required")
		http.Error(w, "UserID, VPNConfig and VPNIP are required", http.StatusBadRequest)
		return
	}

	userID := mustGetUserIDFromContext(r)

	err := db.CreateVPNConfig(vpnCreate.VPNConfig, vpnCreate.VPNIP, userID)
	if err != nil {
		logger.Error("Failed to create VPN config in DB", "error", err)
		http.Error(w, "Failed to create VPN config", http.StatusInternalServerError)
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

	var vpns []internal.VPNUpdate
	for i := range vpnConfigs {
		vpns = append(vpns, internal.VPNUpdate{
			ID:        vpnConfigs[i].ID,
			VPNConfig: vpnConfigs[i].VPNConfig,
			VPNIP:     vpnConfigs[i].VPNIP,
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

func getUserVPNConfig(w http.ResponseWriter, r *http.Request) {
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
