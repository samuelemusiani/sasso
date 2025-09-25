package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/proxmox"

	"github.com/go-chi/chi/v5"
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

type createBackupRequest struct {
	ID uint `json:"id"`
}

func restoreBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := getVMFromContext(r)

	backupid := chi.URLParam(r, "backupid")

	id, err := proxmox.RestoreBackup(uint64(userID), vm.ID, backupid, vm.CreatedAt)
	if err != nil {
		logger.With("userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err).Error("Failed to restore backup")
		http.Error(w, "Failed to restore backup", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(createBackupRequest{ID: id})
	if err != nil {
		logger.With("error", err).Error("Failed to encode restore backup response to JSON")
		http.Error(w, "Failed to encode restore backup response to JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func createBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := getVMFromContext(r)

	backupid := chi.URLParam(r, "backupid")

	id, err := proxmox.CreateBackup(uint64(userID), vm.ID)
	if err != nil {
		logger.With("userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err).Error("Failed to delete backup")
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(createBackupRequest{ID: id})
	if err != nil {
		logger.With("error", err).Error("Failed to encode delete backup response to JSON")
		http.Error(w, "Failed to encode delete backup response to JSON", http.StatusInternalServerError)
		return
	}
}

func deleteBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := getVMFromContext(r)

	backupid := chi.URLParam(r, "backupid")

	id, err := proxmox.DeleteBackup(uint64(userID), vm.ID, backupid, vm.CreatedAt)
	if err != nil {
		logger.With("userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err).Error("Failed to delete backup")
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(createBackupRequest{ID: id})
	if err != nil {
		logger.With("error", err).Error("Failed to encode delete backup response to JSON")
		http.Error(w, "Failed to encode delete backup response to JSON", http.StatusInternalServerError)
		return
	}
}
