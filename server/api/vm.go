package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"samuelemusiani/sasso/server/proxmox"
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
}

type newVMRequest struct {
	Cores                uint `json:"cores"`
	RAM                  uint `json:"ram"`
	Disk                 uint `json:"disk"`
	IncludeGlobalSSHKeys bool `json:"include_global_ssh_keys"`
}

func newVM(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var req newVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	vm, err := proxmox.NewVM(userID, req.Cores, req.RAM, req.Disk, req.IncludeGlobalSSHKeys)
	if err != nil {
		if errors.Is(err, proxmox.ErrInsufficientResources) {
			http.Error(w, "Insufficient resources", http.StatusForbidden)
		} else if errors.Is(err, proxmox.ErrInvalidVMParam) {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

func getVM(w http.ResponseWriter, r *http.Request) {
	vm := getVMFromContext(r)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vm); err != nil {
		logger.With("vmID", vm.ID, "error", err).Error("Failed to encode VM to JSON")
		http.Error(w, "Failed to encode VM to JSON", http.StatusInternalServerError)
		return
	}
}

func deleteVM(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vmID := getVMFromContext(r).ID

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

func changeVMState(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mustGetUserIDFromContext(r)
		vmID := getVMFromContext(r).ID

		var err error
		switch action {
		case "start", "stop", "restart":
			err = proxmox.ChangeVMStatus(userID, vmID, action)
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
			return
		}

		if err != nil {
			logger.With("userID", userID, "vmID", vmID, "action", action, "error", err).Error("Failed to change VM state")
			if errors.Is(err, proxmox.ErrVMNotFound) {
				http.Error(w, "Failed to change VM state", http.StatusNotFound)
			} else if errors.Is(err, proxmox.ErrInvalidVMState) {
				http.Error(w, "Invalid VM state for this action", http.StatusConflict)
			} else {
				http.Error(w, "Failed to change VM state", http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
