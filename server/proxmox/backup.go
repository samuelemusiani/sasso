package proxmox

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"samuelemusiani/sasso/server/db"

	"github.com/luthermonson/go-proxmox"
)

type Backup struct {
	// ID is the Volid hashed
	ID        string    `json:"id"`
	Ctime     time.Time `json:"ctime"`
	CanDelete bool      `json:"can_delete"`
	Name      string    `json:"name"`
	Notes     string    `json:"notes"`
	Protected bool      `json:"protected"`
}

const (
	MAX_BACKUPS_PER_USER           = 2
	MAX_PROTECTED_BACKUPS_PER_USER = 4
)

var (
	BackupRequestStatusPending   = "pending"
	BackupRequestStatusCompleted = "completed"
	BackupRequestStatusFailed    = "failed"

	BackupRequestTypeCreate  = "create"
	BackupRequestTypeRestore = "restore"
	BackupRequestTypeDelete  = "delete"

	BackupSassoString = "sasso-user-backup"

	ErrBackupNotFound             = errors.New("backup_not_found")
	ErrCantDeleteBackup           = errors.New("cant_delete_backup")
	ErrPendingBackupRequest       = errors.New("pending_backup_request")
	ErrMaxBackupsReached          = errors.New("max_backups_reached")
	ErrMaxProtectedBackupsReached = errors.New("max_protected_backups_reached")
	ErrBackupNameTooLong          = errors.New("backup_name_too_long")
	ErrBackupNotesTooLong         = errors.New("backup_notes_too_long")
)

var goodVMStatesForBackupManipulation = []VMStatus{VMStatusRunning, VMStatusStopped, VMStatusSuspended}

func ListBackups(vmID uint64, since time.Time) ([]Backup, error) {
	vm, err := db.GetVMByID(vmID)
	if err != nil {
		logger.Error("failed to get VM by ID", "VMID", vmID, "error", err)
		return nil, err
	}
	if !slices.Contains(goodVMStatesForBackupManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to list backups", "VMID", vmID, "status", vm.Status)
		return nil, ErrInvalidVMState
	}

	_, _, _, mcontent, err := listBackups(vmID, since)
	if err != nil {
		return nil, err
	}

	var backups []Backup
	for _, item := range mcontent {
		h := hmac.New(sha256.New, nonce)
		h.Write([]byte(item.Volid))

		var name, notes string
		var canDelete bool
		bkn, err := parseBackupNotes(item.Notes)
		if err != nil {
			name = "unknown"
			notes = ""
			canDelete = false
		} else {
			name = bkn.Name
			notes = bkn.Notes
			canDelete = bkn.SassoVerifier == BackupSassoString
		}

		backups = append(backups, Backup{
			ID:        hex.EncodeToString(h.Sum(nil)),
			Ctime:     time.Unix(int64(item.Ctime), 0),
			CanDelete: canDelete,
			Name:      name,
			Notes:     notes,
			Protected: bool(item.Protected),
		})
	}

	return backups, nil
}

func CreateBackup(userID uint, groupID *uint, vmID uint64, name, notes string) (uint, error) {
	if len(name) > 40 {
		return 0, ErrBackupNameTooLong
	} else if len(notes)*4/3 > 800 {
		return 0, ErrBackupNotesTooLong
	}

	isPending, err := db.IsAPendingBackupRequest(uint(vmID))
	if err != nil {
		logger.Error("failed to check for pending backup requests", "error", err)
		return 0, err
	}
	if isPending {
		return 0, ErrPendingBackupRequest
	}

	vm, err := db.GetVMByID(vmID)
	if err != nil {
		logger.Error("failed to get VM by ID", "VMID", vmID, "error", err)
		return 0, err
	}
	if !slices.Contains(goodVMStatesForBackupManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to list backups", "VMID", vmID, "status", vm.Status)
		return 0, ErrInvalidVMState
	}

	_, _, _, mcontent, err := listBackups(vmID, time.Time{})
	if err != nil {
		logger.Error("failed to list backups", "error", err)
		return 0, err
	}
	count := 0
	for _, i := range mcontent {
		bkn, err := parseBackupNotes(i.Notes)
		if err != nil {
			continue
		}
		if groupID != nil {
			if bkn.OwnerID == *groupID {
				count++
			}
		} else {
			if bkn.OwnerID == userID {
				count++
			}
		}
	}

	if count >= MAX_BACKUPS_PER_USER {
		return 0, ErrMaxBackupsReached
	}

	var bkr *db.BackupRequest
	if groupID != nil {
		bkr, err = db.NewBackupRequestForGroup(BackupRequestTypeCreate, BackupRequestStatusPending, uint(vmID), *groupID, name, notes)
	} else {
		bkr, err = db.NewBackupRequestForUser(BackupRequestTypeCreate, BackupRequestStatusPending, uint(vmID), userID, name, notes)
	}

	if err != nil {
		logger.Error("failed to create backup request", "error", err)
		return 0, err
	}

	return bkr.ID, nil
}

func DeleteBackup(userID uint, groupID *uint, vmID uint64, backupid string, since time.Time) (uint, error) {
	isPending, err := db.IsAPendingBackupRequest(uint(vmID))
	if err != nil {
		logger.Error("failed to check for pending backup requests", "error", err)
		return 0, err
	}
	if isPending {
		return 0, ErrPendingBackupRequest
	}

	vm, err := db.GetVMByID(vmID)
	if err != nil {
		logger.Error("failed to get VM by ID", "VMID", vmID, "error", err)
		return 0, err
	}
	if !slices.Contains(goodVMStatesForBackupManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to list backups", "VMID", vmID, "status", vm.Status)
		return 0, ErrInvalidVMState
	}

	volid, err := findVolid(vmID, backupid, since, true)
	if err != nil {
		return 0, err
	}

	var bkr *db.BackupRequest
	if groupID != nil {
		bkr, err = db.NewBackupRequestWithVolidForGroup(BackupRequestTypeDelete, BackupRequestStatusPending, &volid, uint(vmID), *groupID, "", "")
	} else {
		bkr, err = db.NewBackupRequestWithVolidForUser(BackupRequestTypeDelete, BackupRequestStatusPending, &volid, uint(vmID), userID, "", "")
	}
	if err != nil {
		logger.Error("failed to create backup request", "error", err)
		return 0, err
	}
	return bkr.ID, nil
}

func RestoreBackup(userID uint, groupID *uint, vmID uint64, backupid string, since time.Time) (uint, error) {
	isPending, err := db.IsAPendingBackupRequest(uint(vmID))
	if err != nil {
		logger.Error("failed to check for pending backup requests", "error", err)
		return 0, err
	}
	if isPending {
		return 0, ErrPendingBackupRequest
	}

	vm, err := db.GetVMByID(vmID)
	if err != nil {
		logger.Error("failed to get VM by ID", "VMID", vmID, "error", err)
		return 0, err
	}
	if !slices.Contains(goodVMStatesForBackupManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to list backups", "VMID", vmID, "status", vm.Status)
		return 0, ErrInvalidVMState
	}

	volid, err := findVolid(vmID, backupid, since, false)
	if err != nil {
		return 0, err
	}

	var bkr *db.BackupRequest
	if groupID != nil {
		bkr, err = db.NewBackupRequestWithVolidForGroup(BackupRequestTypeRestore, BackupRequestStatusPending, &volid, uint(vmID), *groupID, "", "")
	} else {
		bkr, err = db.NewBackupRequestWithVolidForUser(BackupRequestTypeRestore, BackupRequestStatusPending, &volid, uint(vmID), userID, "", "")
	}
	if err != nil {
		logger.Error("failed to create backup request", "error", err)
		return 0, err
	}

	return bkr.ID, nil
}

func listBackups(vmID uint64, since time.Time) (cluster *proxmox.Cluster, node *proxmox.Node, vm *proxmox.VirtualMachine, scontent []*proxmox.StorageContent, err error) {
	cluster, err = getProxmoxCluster(client)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	m, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	nodeName, ok := m[vmID]
	if !ok {
		return nil, nil, nil, nil, ErrVMNotFound
	}

	node, err = getProxmoxNode(client, nodeName)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	s, err := getProxmoxStorage(node, cBackup.Storage)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	mcontent, err := getProxmoxStorageContent(s)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	nContent := make([]*proxmox.StorageContent, 0, len(mcontent))
	for _, item := range mcontent {
		if item.VMID == vmID && time.Unix(int64(item.Ctime), 0).After(since) {
			nContent = append(nContent, item)
		}
	}

	return cluster, node, vm, nContent, nil
}

// Deletion is true if we are looking for a backup to delete, false if we are looking for a backup to restore
func findVolid(vmID uint64, backupid string, since time.Time, deletion bool) (string, error) {
	_, _, _, mcontent, err := listBackups(vmID, since)
	if err != nil {
		return "", err
	}

	for _, item := range mcontent {
		h := hmac.New(sha256.New, nonce)
		h.Write([]byte(item.Volid))

		if hex.EncodeToString(h.Sum(nil)) == backupid {
			bkn, err := parseBackupNotes(item.Notes)
			if err != nil && deletion {
				continue
			}

			if !deletion || bkn.SassoVerifier == BackupSassoString {
				return item.Volid, nil
			} else {
				return "", ErrCantDeleteBackup
			}
		}
	}

	return "", ErrBackupNotFound
}

type BackupNotes struct {
	Name          string `json:"name"`
	Notes         string `json:"notes"`
	OwnerID       uint   `json:"owner_id"`
	OwnerType     string `json:"owner_type"`
	SassoVerifier string `json:"sasso_verifier"`
}

func generateBackNotes(name, notes string, ownerID uint, ownerType string) (string, error) {
	base64Notes := base64.StdEncoding.EncodeToString([]byte(notes))
	bn := BackupNotes{
		Name:          name,
		Notes:         base64Notes,
		OwnerID:       ownerID,
		OwnerType:     ownerType,
		SassoVerifier: BackupSassoString,
	}
	b, err := json.Marshal(bn)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseBackupNotes(notes string) (*BackupNotes, error) {
	var bn BackupNotes
	err := json.Unmarshal([]byte(notes), &bn)
	if err != nil {
		return nil, err
	}
	decodedNotes, err := base64.StdEncoding.DecodeString(bn.Notes)
	if err != nil {
		return nil, err
	}
	bn.Notes = string(decodedNotes)
	return &bn, nil
}

func ProtectBackup(userID, vmID uint64, backupid string, since time.Time, protected bool) (bool, error) {
	vm, err := db.GetVMByID(vmID)
	if err != nil {
		logger.Error("failed to get VM by ID", "VMID", vmID, "error", err)
		return false, err
	}
	if !slices.Contains(goodVMStatesForBackupManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to list backups", "VMID", vmID, "status", vm.Status)
		return false, ErrInvalidVMState
	}

	// We only need to check the upper limit if we are trying to protect a backup
	_, node, _, mcontent, err := listBackups(vmID, since)
	if err != nil {
		logger.Error("failed to list backups", "error", err)
		return false, err
	}

	if protected {
		count := 0
		for _, i := range mcontent {
			if bool(i.Protected) {
				count++
			}
		}

		if count >= MAX_PROTECTED_BACKUPS_PER_USER {
			return false, ErrMaxProtectedBackupsReached
		}
	}

	// TODO: optimize this, we search the backup twice
	volid, err := findVolid(vmID, backupid, since, false)
	if err != nil {
		return false, err
	}

	pending, err := db.IsAPendingBackupRequestWithVolid(uint(vmID), volid)
	if err != nil {
		logger.Error("failed to check for pending backup requests", "error", err)
		return false, err
	}
	if pending {
		return false, ErrPendingBackupRequest
	}

	s, err := getProxmoxStorage(node, cBackup.Storage)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	isSuccessful, err := s.ChangeProtection(ctx, protected, volid)
	cancel()
	if err != nil {
		logger.Error("Failed to change backup protection", "error", err)
		return false, err
	}
	if !isSuccessful {
		logger.Error("Failed to change backup protection: operation not successful")
		return false, errors.New("operation_not_successful")
	}
	return true, nil
}
