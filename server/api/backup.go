package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
	"strconv"
	"time"

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
}

type createBackupRequest struct {
	ID uint `json:"id"`
}

func restoreBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := getVMFromContext(r)

	if vm.Status != string(proxmox.VMStatusStopped) {
		http.Error(w, "VM must be stopped to restore a backup", http.StatusBadRequest)
		return
	}

	backupid := chi.URLParam(r, "backupid")

	id, err := proxmox.RestoreBackup(uint64(userID), vm.ID, backupid, vm.CreatedAt)
	if err != nil {
		if err == proxmox.ErrBackupNotFound {
			http.Error(w, "Backup not found", http.StatusNotFound)
			return
		} else if err == proxmox.ErrPendingBackupRequest {
			http.Error(w, "There is already a pending backup request for this VM", http.StatusBadRequest)
			return
		}
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
}

type CreateBackupRequestBody struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

func createBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := getVMFromContext(r)

	backupid := chi.URLParam(r, "backupid")

	var reqBody CreateBackupRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, err := proxmox.CreateBackup(uint64(userID), vm.ID, reqBody.Name, reqBody.Notes)
	if err != nil {
		if err == proxmox.ErrPendingBackupRequest {
			http.Error(w, "There is already a pending backup request for this VM", http.StatusBadRequest)
			return
		} else if err == proxmox.ErrMaxBackupsReached {
			http.Error(w, "Maximum number of backups reached for this user", http.StatusBadRequest)
			return
		} else if err == proxmox.ErrBackupNameTooLong {
			http.Error(w, "Backup name too long", http.StatusBadRequest)
			return
		} else if err == proxmox.ErrBackupNotesTooLong {
			http.Error(w, "Backup notes too long", http.StatusBadRequest)
			return
		}
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
		if err == proxmox.ErrBackupNotFound {
			http.Error(w, "Backup not found", http.StatusNotFound)
			return
		} else if err == proxmox.ErrCantDeleteBackup {
			http.Error(w, "Can't delete backup", http.StatusBadRequest)
			return
		} else if err == proxmox.ErrPendingBackupRequest {
			http.Error(w, "Pending backup request", http.StatusBadRequest)
			return
		}

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

type BackupRequest struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Type   string `json:"type"`
	Status string `json:"status"`
	VMID   uint   `json:"vm_id"`
}

func listBackupRequests(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	bkr, err := db.GetBackupRequestsByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to list backup requests")
		http.Error(w, "Failed to list backup requests", http.StatusInternalServerError)
		return
	}

	resp := make([]BackupRequest, 0, len(bkr))
	for _, b := range bkr {
		resp = append(resp, BackupRequest{
			ID:        b.ID,
			CreatedAt: b.CreatedAt,
			Type:      b.Type,
			Status:    b.Status,
			VMID:      b.VMID,
		})
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to encode backup requests to JSON")
		http.Error(w, "Failed to encode backup requests to JSON", http.StatusInternalServerError)
		return
	}
}

func getBackupRequest(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	sbkrID := chi.URLParam(r, "requestid")

	bkrID, err := strconv.ParseUint(sbkrID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid backup request ID", http.StatusBadRequest)
		return
	}

	bkr, err := db.GetBackupRequestByIDAndUserID(uint(bkrID), userID)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Backup request not found", http.StatusNotFound)
			return
		}
		logger.With("userID", userID, "bkrID", bkrID, "error", err).Error("Failed to get backup request")
		http.Error(w, "Failed to get backup request", http.StatusInternalServerError)
		return
	}

	resp := BackupRequest{
		ID:        bkr.ID,
		CreatedAt: bkr.CreatedAt,
		Type:      bkr.Type,
		Status:    bkr.Status,
		VMID:      bkr.VMID,
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.With("userID", userID, "bkrID", bkrID, "error", err).Error("Failed to encode backup request to JSON")
		http.Error(w, "Failed to encode backup request to JSON", http.StatusInternalServerError)
		return
	}
}

type ProtectBackupRequest struct {
	Protected bool `json:"protected"`
}

func protectBackup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	vm := getVMFromContext(r)

	backupid := chi.URLParam(r, "backupid")

	var reqBody ProtectBackupRequest
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	protected, err := proxmox.ProtectBackup(uint64(userID), vm.ID, backupid, vm.CreatedAt, reqBody.Protected)
	if err != nil {
		if err == proxmox.ErrBackupNotFound {
			http.Error(w, "Backup not found", http.StatusNotFound)
			return
		} else if err == proxmox.ErrMaxProtectedBackupsReached {
			http.Error(w, "Max protected backups reached", http.StatusBadRequest)
			return
		}
		logger.With("userID", userID, "vmID", vm.ID, "backupid", backupid, "error", err).Error("Failed to protect backup")
		http.Error(w, "Failed to protect backup", http.StatusInternalServerError)
		return
	}
	if !protected {
		http.Error(w, "Failed to protect backup", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
