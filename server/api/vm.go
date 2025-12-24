package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
)

func vms(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

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
}

type newVMRequest struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
	Cores uint   `json:"cores"`
	RAM   uint   `json:"ram"`
	Disk  uint   `json:"disk"`
	// Number of months the VM should live
	LifeTime             uint `json:"lifetime"`
	IncludeGlobalSSHKeys bool `json:"include_global_ssh_keys"`

	GroupID *uint `json:"group_id,omitempty"`
}

func newVM(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var req newVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	m := getUserResourceMutex(userID)

	m.Lock()
	defer m.Unlock()

	newVMRequest := proxmox.NewVMRequest{
		Name:                 req.Name,
		Notes:                req.Notes,
		Cores:                req.Cores,
		RAM:                  req.RAM,
		Disk:                 req.Disk,
		LifeTime:             req.LifeTime,
		IncludeGlobalSSHKeys: req.IncludeGlobalSSHKeys,
	}

	ownerID := userID
	ownerType := proxmox.OwnerTypeUser

	if req.GroupID != nil {
		ownerID = *req.GroupID
		ownerType = proxmox.OwnerTypeGroup
	}

	vm, err := proxmox.NewVM(ownerType, ownerID, userID, newVMRequest)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrInsufficientResources):
			http.Error(w, "Insufficient resources", http.StatusForbidden)
		case errors.Is(err, proxmox.ErrInvalidVMParam):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			logger.Error("Failed to create new VM", "userID", userID, "error", err)
			http.Error(w, "Failed to create new VM", http.StatusInternalServerError)
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(vm); err != nil {
		logger.Error("Failed to encode new VM to JSON", "error", err)
		http.Error(w, "Failed to encode new VM to JSON", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func getVM(w http.ResponseWriter, r *http.Request) {
	vm := mustGetVMFromContext(r)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(vm); err != nil {
		logger.Error("Failed to encode VM to JSON", "vmID", vm.ID, "error", err)
		http.Error(w, "Failed to encode VM to JSON", http.StatusInternalServerError)

		return
	}
}

func deleteVM(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := mustGetVMFromContext(r)
	vmID := vm.ID

	ownerID := userID
	owerType := proxmox.OwnerTypeUser

	if vm.OwnerType == "Group" {
		ownerID = vm.OwnerID
		owerType = proxmox.OwnerTypeGroup
	}

	m := getVMMutex(uint(vmID))

	m.Lock()
	defer m.Unlock()

	bkPending, err := db.IsAPendingBackupRequest(uint(vmID))
	if err != nil {
		logger.Error("Failed to check for pending backup requests", "vmID", vmID, "error", err)
		http.Error(w, "Failed to delete VM", http.StatusInternalServerError)

		return
	}

	if bkPending {
		http.Error(w, "Cannot delete VM with pending backup requests", http.StatusConflict)

		return
	}

	m2 := getUserResourceMutex(userID)

	m2.Lock()
	defer m2.Unlock()

	if err := proxmox.DeleteVM(owerType, ownerID, userID, vm.ID); err != nil {
		logger.Error("Failed to delete VM", "userID", userID, "vmID", vmID, "error", err)

		switch {
		case errors.Is(err, proxmox.ErrVMNotFound):
			http.Error(w, "Failed to delete VM", http.StatusNotFound)
		case errors.Is(err, proxmox.ErrPermissionDenied):
			http.Error(w, "Permission denied", http.StatusForbidden)
		default:
			http.Error(w, "Failed to delete VM", http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func changeVMState(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mustGetUserIDFromContext(r)
		vm := mustGetVMFromContext(r)
		vmID := vm.ID

		if vm.LifeTime.Before(time.Now()) {
			http.Error(w, "Cannot change state of expired VM", http.StatusForbidden)

			return
		}

		ownerID := userID
		ownerType := proxmox.OwnerTypeUser

		if vm.OwnerType == "Group" {
			ownerID = vm.OwnerID
			ownerType = proxmox.OwnerTypeGroup
		}

		m := getVMMutex(uint(vmID))

		m.Lock()
		defer m.Unlock()

		// The restore action cannot be performed if the VM is running. We must
		// the change of state.
		bkRequests, err := db.GetBackupRequestsByVMIDStatusAndType(uint(vmID), "pending", "restore")
		if err != nil {
			logger.Error("Failed to check for pending backup requests", "vmID", vmID, "error", err)
			http.Error(w, "Failed to delete VM", http.StatusInternalServerError)

			return
		}

		if len(bkRequests) > 0 {
			http.Error(w, "Cannot update VM status with pending restore backup requests", http.StatusConflict)

			return
		}

		switch action {
		case "start", "stop", "restart":
			err = proxmox.ChangeVMStatus(r.Context(), ownerType, ownerID, userID, vm.ID, action)
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)

			return
		}

		if err != nil {
			logger.Error("Failed to change VM state", "userID", userID, "vmID", vmID, "action", action, "error", err)

			switch {
			case errors.Is(err, proxmox.ErrVMNotFound):
				http.Error(w, "Failed to change VM state", http.StatusNotFound)
			case errors.Is(err, proxmox.ErrInvalidVMState):
				http.Error(w, "Invalid VM state for this action", http.StatusConflict)
			case errors.Is(err, proxmox.ErrPermissionDenied):
				http.Error(w, "Permission denied", http.StatusForbidden)
			default:
				http.Error(w, "Failed to change VM state", http.StatusInternalServerError)
			}

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type updateVMLifetimeRequest struct {
	// Number of months to extend the VM lifetime
	ExtendBy uint `json:"extend_by"`
}

func updateVMLifetime(w http.ResponseWriter, r *http.Request) {
	var request updateVMLifetimeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	vmID := mustGetVMFromContext(r).ID

	m := getVMMutex(uint(vmID))

	m.Lock()
	defer m.Unlock()

	err := proxmox.UpdateVMLifetime(vmID, request.ExtendBy)
	if err != nil {
		if errors.Is(err, proxmox.ErrInvalidVMParam) {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		logger.Error("Failed to update VM lifetime", "vmID", vmID, "error", err)
		http.Error(w, "Failed to update VM lifetime", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateResourcesRequest struct {
	Cores uint `json:"cores"`
	RAM   uint `json:"ram"`
	Disk  uint `json:"disk"`
}

func updateVMResources(w http.ResponseWriter, r *http.Request) {
	var request updateResourcesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	vm := mustGetVMFromContext(r)

	if vm.LifeTime.Before(time.Now()) {
		http.Error(w, "Cannot update resources of expired VM", http.StatusForbidden)

		return
	}

	vmid := vm.ID
	if vm.OwnerType == "Group" {
		role := mustGetUserRoleInGroupFromContext(r)
		if role != "admin" && role != "owner" {
			http.Error(w, "Permission denied", http.StatusForbidden)

			return
		}
	}

	userID := mustGetUserIDFromContext(r)

	m := getVMMutex(uint(vmid))

	m.Lock()
	defer m.Unlock()

	m2 := getUserResourceMutex(userID)

	m2.Lock()
	defer m2.Unlock()

	err := proxmox.UpdateVMResources(vmid, request.Cores, request.RAM, request.Disk)
	if err != nil {
		if errors.Is(err, proxmox.ErrInsufficientResources) {
			http.Error(w, "Insufficient resources", http.StatusForbidden)

			return
		} else if errors.Is(err, proxmox.ErrInvalidVMParam) {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		logger.Error("Failed to update VM resources", "vmID", vmid, "error", err)
		http.Error(w, "Failed to update VM resources", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
