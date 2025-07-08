package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/proxmox"
)

func vms(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	vms, err := proxmox.GetVMsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get VMs", "userID", userID, "error", err)
		http.Error(w, "Failed to get VMs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vms); err != nil {
		logger.Error("Failed to encode VMs to JSON", "error", err)
		http.Error(w, "Failed to encode VMs to JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
