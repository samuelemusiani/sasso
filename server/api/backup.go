package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"samuelemusiani/sasso/server/proxmox"
)

func listBackups(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := mustGetVMFromContext(r)

	bks, err := proxmox.ListBackups(r.Context(), vm.ID, vm.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrVMNotFound):
			http.Error(w, "VM not found", http.StatusNotFound)

			return
		case errors.Is(err, proxmox.ErrInvalidVMState):
			http.Error(w, "Invalid VM state", http.StatusConflict)

			return
		}

		logger.Error("Failed to list backups", "userID", userID, "vmID", vm.ID, "error", err)
		http.Error(w, "Failed to list backups", http.StatusInternalServerError)

		return
	}

	err = json.NewEncoder(w).Encode(bks)
	if err != nil {
		logger.Error("Failed to encode backups to JSON", "userID", userID, "vmID", vm.ID, "error", err)
		http.Error(w, "Failed to encode backups to JSON", http.StatusInternalServerError)

		return
	}
}

type createBackupRequest struct {
	ID uint `json:"id"`
}

func restoreBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := mustGetVMFromContext(r)

	if vm.LifeTime.Before(time.Now()) {
		http.Error(w, "Cannot restore backup in expired VM", http.StatusConflict)

		return
	}

	m := getVMMutex(uint(vm.ID))

	m.Lock()
	defer m.Unlock()

	var groupID *uint

	if vm.OwnerType == "Group" {
		role := mustGetUserRoleInGroupFromContext(r)
		if role == "member" {
			http.Error(w, "Only group admins can restore backups", http.StatusForbidden)

			return
		}

		tmp := mustGetGroupIDFromContext(r)
		groupID = &tmp
	}

	if vm.Status != string(proxmox.VMStatusStopped) {
		http.Error(w, "VM must be stopped to restore a backup", http.StatusBadRequest)

		return
	}

	backupid := chi.URLParam(r, "backupid")

	id, err := proxmox.RestoreBackup(r.Context(), userID, groupID, vm.ID, backupid, vm.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrBackupNotFound):
			http.Error(w, "Backup not found", http.StatusNotFound)

			return
		case errors.Is(err, proxmox.ErrPendingBackupRequest):
			http.Error(w, "There is already a pending backup request for this VM", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrInvalidVMState):
			http.Error(w, "Invalid VM state", http.StatusConflict)

			return
		}

		logger.Error("Failed to restore backup", "userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err)
		http.Error(w, "Failed to restore backup", http.StatusInternalServerError)

		return
	}

	err = json.NewEncoder(w).Encode(createBackupRequest{ID: id})
	if err != nil {
		logger.Error("Failed to encode restore backup response to JSON", "error", err)
		http.Error(w, "Failed to encode restore backup response to JSON", http.StatusInternalServerError)

		return
	}
}

type CreateBackupRequestBody struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

func createBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := mustGetVMFromContext(r)

	if vm.LifeTime.Before(time.Now()) {
		http.Error(w, "Cannot create backup in expired VM", http.StatusConflict)

		return
	}

	var groupID *uint

	if vm.OwnerType == "Group" {
		tmp := mustGetGroupIDFromContext(r)
		groupID = &tmp

		role := mustGetUserRoleInGroupFromContext(r)
		if role == "member" {
			http.Error(w, "Only group admins can create backups", http.StatusForbidden)

			return
		}
	}

	backupid := chi.URLParam(r, "backupid")

	var reqBody CreateBackupRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	if reqBody.Name == "" {
		http.Error(w, "Backup name is required", http.StatusBadRequest)

		return
	}

	m := getVMMutex(uint(vm.ID))

	m.Lock()
	defer m.Unlock()

	id, err := proxmox.CreateBackup(r.Context(), userID, groupID, vm.ID, reqBody.Name, reqBody.Notes)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrPendingBackupRequest):
			http.Error(w, "There is already a pending backup request for this VM", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrMaxBackupsReached):
			http.Error(w, "Maximum number of backups reached for this user", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrBackupNameTooLong):
			http.Error(w, "Backup name too long", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrBackupNotesTooLong):
			http.Error(w, "Backup notes too long", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrInvalidVMState):
			http.Error(w, "Invalid VM state", http.StatusConflict)

			return
		}

		logger.Error("Failed to delete backup", "userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err)
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)

		return
	}

	err = json.NewEncoder(w).Encode(createBackupRequest{ID: id})
	if err != nil {
		logger.Error("Failed to encode delete backup response to JSON", "error", err)
		http.Error(w, "Failed to encode delete backup response to JSON", http.StatusInternalServerError)

		return
	}
}

func deleteBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := mustGetVMFromContext(r)

	if vm.LifeTime.Before(time.Now()) {
		http.Error(w, "Cannot delete backup in expired VM", http.StatusConflict)

		return
	}

	var groupID *uint

	if vm.OwnerType == "Group" {
		role := mustGetUserRoleInGroupFromContext(r)
		if role == "member" {
			http.Error(w, "Only group admins can delete backups", http.StatusForbidden)

			return
		}

		tmp := mustGetGroupIDFromContext(r)
		groupID = &tmp
	}

	backupid := chi.URLParam(r, "backupid")

	id, err := proxmox.DeleteBackup(r.Context(), userID, groupID, vm.ID, backupid, vm.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrBackupNotFound):
			http.Error(w, "Backup not found", http.StatusNotFound)

			return
		case errors.Is(err, proxmox.ErrCantDeleteBackup):
			http.Error(w, "Can't delete backup", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrPendingBackupRequest):
			http.Error(w, "Pending backup request", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrInvalidVMState):
			http.Error(w, "Invalid VM state", http.StatusConflict)

			return
		}

		logger.Error("Failed to delete backup", "userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err)
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)

		return
	}

	err = json.NewEncoder(w).Encode(createBackupRequest{ID: id})
	if err != nil {
		logger.Error("Failed to encode delete backup response to JSON", "error", err)
		http.Error(w, "Failed to encode delete backup response to JSON", http.StatusInternalServerError)

		return
	}
}

func listBackupRequests(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	l := logger.With("userID", userID)

	var groupID *uint

	if mustGetVMFromContext(r).OwnerType == "Group" {
		tmp := mustGetGroupIDFromContext(r)
		groupID = &tmp
		l = l.With("groupID", *groupID)
	}

	var (
		bkr []proxmox.ReturnBackupRequest
		err error
	)

	status := r.URL.Query().Get("status")
	svmid := chi.URLParam(r, "vmid")

	vmid, err := strconv.ParseUint(svmid, 10, 32)
	if err != nil {
		http.Error(w, "Invalid VM ID", http.StatusBadRequest)

		return
	}

	if groupID != nil {
		bkr, err = proxmox.GetBackupRequestsByGroupID(*groupID, status, uint(vmid))
	} else {
		bkr, err = proxmox.GetBackupRequestsByUserID(userID, status, uint(vmid))
	}

	if err != nil {
		l.Error("Failed to list backup requests", "error", err)
		http.Error(w, "Failed to list backup requests", http.StatusInternalServerError)

		return
	}

	err = json.NewEncoder(w).Encode(bkr)
	if err != nil {
		l.Error("Failed to encode backup requests to JSON", "error", err)
		http.Error(w, "Failed to encode backup requests to JSON", http.StatusInternalServerError)

		return
	}
}

func getBackupRequest(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	sbkrID := chi.URLParam(r, "requestid")

	bkrID, err := strconv.ParseUint(sbkrID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid backup request ID", http.StatusBadRequest)

		return
	}

	bkr, err := proxmox.GetBackupRequestByID(uint(bkrID))
	if err != nil {
		if errors.Is(err, proxmox.ErrNotFound) {
			http.Error(w, "Backup request not found", http.StatusNotFound)

			return
		}

		logger.Error("Failed to get backup request", "userID", userID, "bkrID", bkrID, "error", err)
		http.Error(w, "Failed to get backup request", http.StatusInternalServerError)

		return
	}

	if mustGetVMFromContext(r).OwnerType == "Group" {
		if bkr.OwnerType != "Group" || bkr.OwnerID != mustGetGroupIDFromContext(r) {
			http.Error(w, "Backup request not found", http.StatusNotFound)

			return
		}
	} else {
		if bkr.OwnerType != "User" || bkr.OwnerID != userID {
			http.Error(w, "Backup request not found", http.StatusNotFound)

			return
		}
	}

	err = json.NewEncoder(w).Encode(bkr)
	if err != nil {
		logger.Error("Failed to encode backup request to JSON", "userID", userID, "bkrID", bkrID, "error", err)
		http.Error(w, "Failed to encode backup request to JSON", http.StatusInternalServerError)

		return
	}
}

type protectBackupRequest struct {
	Protected bool `json:"protected"`
}

func protectBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := mustGetVMFromContext(r)

	if vm.OwnerType == "Group" {
		role := mustGetUserRoleInGroupFromContext(r)
		if role == "member" {
			http.Error(w, "Only group admins can protect backups", http.StatusForbidden)

			return
		}
	}

	backupid := chi.URLParam(r, "backupid")

	var reqBody protectBackupRequest

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	protected, err := proxmox.ProtectBackup(r.Context(), uint64(userID), vm.ID, backupid, vm.CreatedAt, reqBody.Protected)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrBackupNotFound):
			http.Error(w, "Backup not found", http.StatusNotFound)

			return
		case errors.Is(err, proxmox.ErrMaxProtectedBackupsReached):
			http.Error(w, "Max protected backups reached", http.StatusBadRequest)

			return
		case errors.Is(err, proxmox.ErrInvalidVMState):
			http.Error(w, "Invalid VM state", http.StatusConflict)

			return
		}

		logger.Error("Failed to protect backup", "userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err)
		http.Error(w, "Failed to protect backup", http.StatusInternalServerError)

		return
	}

	if !protected {
		http.Error(w, "Failed to protect backup", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
