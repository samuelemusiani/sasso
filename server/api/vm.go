package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"samuelemusiani/sasso/server/proxmox"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func vms(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	vms, err := proxmox.GetVMsByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get VMs")
		http.Error(w, "Failed to get VMs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vms); err != nil {
		logger.With("error", err).Error("Failed to encode VMs to JSON")
		http.Error(w, "Failed to encode VMs to JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type newVMRequest struct {
	Cores uint `json:"cores"`
	RAM   uint `json:"ram"`
	Disk  uint `json:"disk"`
}

func newVM(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var req newVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	vm, err := proxmox.NewVM(userID, req.Cores, req.RAM, req.Disk)
	if err != nil {
		if errors.Is(err, proxmox.ErrInsufficientResources) {
			http.Error(w, "Insufficient resources", http.StatusForbidden)
		} else {
			logger.With("userID", userID, "error", err).Error("Failed to create new VM")
			http.Error(w, "Failed to create new VM", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vm); err != nil {
		logger.With("error", err).Error("Failed to encode new VM to JSON")
		http.Error(w, "Failed to encode new VM to JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func deleteVM(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	svmID := chi.URLParam(r, "id")

	vmID, err := strconv.ParseUint(svmID, 10, 64)
	if err != nil {
		logger.With("userID", userID, "vmID", svmID, "error", err).Error("Invalid VM ID format")
		http.Error(w, "Invalid VM ID format", http.StatusBadRequest)
		return
	}

	if err := proxmox.DeleteVM(userID, vmID); err != nil {
		logger.With("userID", userID, "vmID", vmID, "error", err).Error("Failed to delete VM")
		if errors.Is(err, proxmox.ErrVMNotFound) {
			http.Error(w, "Failed to delete VM", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete VM", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
