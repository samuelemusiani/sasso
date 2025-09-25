package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/proxmox"
)

func listBackups(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := getVMFromContext(r)

	bks, err := proxmox.ListBackups(vm.ID, vm.CreatedAt)
	if err != nil {
		logger.With("userID", userID, "vmID", vm.ID, "error", err).Error("Failed to list backups")
		http.Error(w, "Failed to list backups", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(bks)
	if err != nil {
		logger.With("userID", userID, "vmID", vm.ID, "error", err).Error("Failed to encode backups to JSON")
		http.Error(w, "Failed to encode backups to JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
