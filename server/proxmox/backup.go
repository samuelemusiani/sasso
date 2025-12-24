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

	"github.com/luthermonson/go-proxmox"
	"samuelemusiani/sasso/server/db"
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
	maxBackupsPerUser          = 2
	maxProtectedBackupsPerUser = 4
)

// volidAction represents the action to be performed on a volid
type volidAction int

const (
	// volidActionSearch simply searches for the volid
	volidActionSearch volidAction = iota
	// volidActionDelete searches the volid and ensures it can be deleted
	volidActionDelete
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

var goodVMStatesForBackupManipulation = []VMStatus{VMStatusRunning, VMStatusStopped, VMStatusPaused}

func ListBackups(parentCtx context.Context, vmID uint64, since time.Time) ([]Backup, error) {
	vm, err := db.GetVMByID(vmID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrVMNotFound
		}

		logger.Error("failed to get VM by ID", "VMID", vmID, "error", err)

		return nil, err
	}

	if !slices.Contains(goodVMStatesForBackupManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to list backups", "VMID", vmID, "status", vm.Status)

		return nil, ErrInvalidVMState
	}

	_, mcontent, err := fetchBackups(parentCtx, vmID, since)
	if err != nil {
		return nil, err
	}

	backups := make([]Backup, 0, len(mcontent))
	for _, item := range mcontent {
		h := hmac.New(sha256.New, nonce)
		h.Write([]byte(item.Volid))

		var (
			name, notes string
			canDelete   bool
		)

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

func CreateBackup(parentCtx context.Context, userID uint, groupID *uint, vmID uint64, name, notes string) (uint, error) {
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

	_, mcontent, err := fetchBackups(parentCtx, vmID, time.Time{})
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

	if count >= maxBackupsPerUser {
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

func DeleteBackup(parentCtx context.Context, userID uint, groupID *uint, vmID uint64, backupid string, since time.Time) (uint, error) {
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

	volid, err := findVolid(parentCtx, vmID, backupid, since, volidActionDelete, nil)
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

func RestoreBackup(parentCtx context.Context, userID uint, groupID *uint, vmID uint64, backupid string, since time.Time) (uint, error) {
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

	volid, err := findVolid(parentCtx, vmID, backupid, since, volidActionSearch, nil)
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

// Proxmox node is returned only for optimization purposes as often the caller
// needs it right after calling this function.
func fetchBackups(parentCtx context.Context, vmID uint64, since time.Time) (node *proxmox.Node, scontent []*proxmox.StorageContent, err error) {
	cluster, err := getProxmoxCluster(parentCtx, client)
	if err != nil {
		return nil, nil, err
	}

	m, err := mapVMIDToProxmoxNodes(parentCtx, cluster)
	if err != nil {
		return nil, nil, err
	}

	nodeName, ok := m[vmID]
	if !ok {
		return nil, nil, ErrVMNotFound
	}

	node, err = getProxmoxNode(parentCtx, client, nodeName)
	if err != nil {
		return nil, nil, err
	}

	s, err := getProxmoxStorage(parentCtx, node, cBackup.Storage)
	if err != nil {
		return nil, nil, err
	}

	mcontent, err := getProxmoxStorageBackups(parentCtx, s, uint(vmID))
	if err != nil {
		return nil, nil, err
	}

	nContent := make([]*proxmox.StorageContent, 0, len(mcontent))
	// As the content returns everything, we need to filter only the backups that
	// actually belong to the VM.
	// The time check is done to avoid listing old backups (before the VM was assigned to the user)
	// as the VMID can be reused.
	for _, item := range mcontent {
		if time.Unix(int64(item.Ctime), 0).After(since) {
			nContent = append(nContent, item)
		}
	}

	return node, nContent, nil
}

// findVolid looks for the volid of a backup given its hashed ID. The backup
// must be related to the given VM ID and must be created after the given time.
// If action is volidActionDelete, it also checks if the backup can be deleted.
// If mcontent is nil, it will be fetched inside the function. Mcontent can be
// provided to optimize multiple calls to this function.
func findVolid(parentCtx context.Context, vmID uint64, backupid string, since time.Time, action volidAction, mcontent []*proxmox.StorageContent) (string, error) {
	if mcontent == nil {
		var err error

		_, mcontent, err = fetchBackups(parentCtx, vmID, since)
		if err != nil {
			return "", err
		}
	}

	for _, item := range mcontent {
		h := hmac.New(sha256.New, nonce)
		h.Write([]byte(item.Volid))

		if hex.EncodeToString(h.Sum(nil)) == backupid {
			bkn, err := parseBackupNotes(item.Notes)
			if err != nil && action == volidActionDelete {
				continue
			}

			if action != volidActionDelete || bkn.SassoVerifier == BackupSassoString {
				return item.Volid, nil
			}

			return "", ErrCantDeleteBackup
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

//nolint:revive // protected is fine here, we're not considering it as a control flag.
func ProtectBackup(parentCtx context.Context, userID, vmID uint64, backupid string, since time.Time, protected bool) (bool, error) {
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
	node, mcontent, err := fetchBackups(parentCtx, vmID, since)
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

		if count >= maxProtectedBackupsPerUser {
			return false, ErrMaxProtectedBackupsReached
		}
	}

	volid, err := findVolid(parentCtx, vmID, backupid, since, volidActionSearch, mcontent)
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

	s, err := getProxmoxStorage(parentCtx, node, cBackup.Storage)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
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
