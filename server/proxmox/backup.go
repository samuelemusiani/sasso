package proxmox

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"samuelemusiani/sasso/server/db"
	"strings"
	"time"

	"github.com/luthermonson/go-proxmox"
)

type Backup struct {
	// ID is the Volid hashed
	ID        string    `json:"id"`
	Ctime     time.Time `json:"ctime"`
	CanDelete bool      `json:"can_delete"`
	Name      string    `json:"name"`
	Notes     string    `json:"notes"`
}

var (
	BackupRequestStatusPending   = "pending"
	BackupRequestStatusCompleted = "completed"
	BackupRequestStatusFailed    = "failed"

	BackupRequestTypeCreate  = "create"
	BackupRequestTypeRestore = "restore"
	BackupRequestTypeDelete  = "delete"

	BackupNoteString = "sasso-user-backup"

	ErrBackupNotFound       = errors.New("backup_not_found")
	ErrCantDeleteBackup     = errors.New("cant_delete_backup")
	ErrPendingBackupRequest = errors.New("pending_backup_request")
)

func ListBackups(vmID uint64, since time.Time) ([]Backup, error) {
	_, _, _, mcontent, err := listBackups(vmID, since)
	if err != nil {
		return nil, err
	}

	var backups []Backup
	for _, item := range mcontent {
		h := hmac.New(sha256.New, nonce)
		h.Write([]byte(item.Volid))

		var name, notes string
		bkn, err := parseBackupNotes(item.Notes)
		if err != nil {
			logger.Warn("failed to parse backup notes, skipping", "error", err)
			name = "unknown"
			notes = ""
		} else {
			name = bkn.Name
			notes = bkn.Notes
		}

		backups = append(backups, Backup{
			ID:        hex.EncodeToString(h.Sum(nil)),
			Ctime:     time.Unix(int64(item.Ctime), 0),
			CanDelete: strings.Contains(item.Notes, BackupNoteString),
			Name:      name,
			Notes:     notes,
		})
	}

	return backups, nil
}

func CreateBackup(userID, vmID uint64, name, notes string) (uint, error) {
	isPending, err := db.IsAPendingBackupRequest(uint(vmID))
	if err != nil {
		logger.Error("failed to check for pending backup requests", "error", err)
		return 0, err
	}
	if isPending {
		return 0, ErrPendingBackupRequest
	}

	bkr, err := db.NewBackupRequest(BackupRequestTypeCreate, BackupRequestStatusPending, uint(vmID), uint(userID), name, notes)
	if err != nil {
		logger.Error("failed to create backup request", "error", err)
		return 0, err
	}

	return bkr.ID, nil
}

func DeleteBackup(userID, vmID uint64, backupid string, since time.Time) (uint, error) {
	isPending, err := db.IsAPendingBackupRequest(uint(vmID))
	if err != nil {
		logger.Error("failed to check for pending backup requests", "error", err)
		return 0, err
	}
	if isPending {
		return 0, ErrPendingBackupRequest
	}

	volid, err := findVolid(vmID, backupid, since, true)
	if err != nil {
		return 0, err
	}

	bkr, err := db.NewBackupRequestWithVolid(BackupRequestTypeDelete, BackupRequestStatusPending, &volid, uint(vmID), uint(userID), "", "")
	if err != nil {
		logger.Error("failed to create backup request", "error", err)
		return 0, err
	}
	return bkr.ID, nil
}

func RestoreBackup(userID, vmID uint64, backupid string, since time.Time) (uint, error) {
	isPending, err := db.IsAPendingBackupRequest(uint(vmID))
	if err != nil {
		logger.Error("failed to check for pending backup requests", "error", err)
		return 0, err
	}
	if isPending {
		return 0, ErrPendingBackupRequest
	}

	volid, err := findVolid(vmID, backupid, since, false)
	if err != nil {
		return 0, err
	}

	bkr, err := db.NewBackupRequestWithVolid(BackupRequestTypeRestore, BackupRequestStatusPending, &volid, uint(vmID), uint(userID), "", "")
	if err != nil {
		logger.Error("failed to create backup request", "error", err)
		return 0, err
	}

	return bkr.ID, nil
}

func listBackups(vmID uint64, since time.Time) (cluster *proxmox.Cluster, node *proxmox.Node, vm *proxmox.VirtualMachine, scontent []*proxmox.StorageContent, err error) {
	cluster, err = getProxmoxCluster(client)
	if err != nil {
		logger.Error("failed to get proxmox cluster", "error", err)
		return nil, nil, nil, nil, err
	}

	m, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.Error("failed to map VMID to Proxmox nodes", "error", err)
		return nil, nil, nil, nil, err
	}

	nodeName, ok := m[vmID]
	if !ok {
		logger.Error("no Proxmox node found for VMID", "vmID", vmID)
		return nil, nil, nil, nil, ErrVMNotFound
	}

	node, err = getProxmoxNode(client, nodeName)
	if err != nil {
		logger.Error("failed to get proxmox node", "error", err)
		return nil, nil, nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s, err := node.Storage(ctx, cBackup.Storage)
	defer cancel()
	if err != nil {
		logger.Error("failed to get storage info", "error", err)
		return nil, nil, nil, nil, err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	mcontent, err := s.GetContent(ctx)
	defer cancel()
	if err != nil {
		logger.Error("failed to get storage content", "error", err)
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
			if !deletion || strings.Contains(item.Notes, BackupNoteString) {
				return item.Volid, nil
			} else {
				return "", ErrCantDeleteBackup
			}
		}
	}

	return "", ErrBackupNotFound
}

type BackupNotes struct {
	Name   string `json:"name"`
	Notes  string `json:"notes"`
	UserID uint   `json:"user_id"`
}

func generateBackNotes(name, notes string, userID uint) (string, error) {
	bn := BackupNotes{
		Name:   name,
		Notes:  notes,
		UserID: userID,
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
	return &bn, nil
}
