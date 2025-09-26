package proxmox

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"samuelemusiani/sasso/server/db"
	"strings"
	"time"

	"github.com/luthermonson/go-proxmox"
)

type Backup struct {
	// Name is the Volid hashed
	Name  string    `json:"name"`
	Ctime time.Time `json:"ctime"`
}

var (
	BackupRequestStatusPending   = "pending"
	BackupRequestStatusCompleted = "completed"
	BackupRequestStatusFailed    = "failed"

	BackupRequestTypeCreate  = "create"
	BackupRequestTypeRestore = "restore"
	BackupRequestTypeDelete  = "delete"

	BackupNoteString = "sasso-user-backup"

	ErrBackupNotFound = errors.New("backup_not_found")
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

		backups = append(backups, Backup{
			Name:  hex.EncodeToString(h.Sum(nil)),
			Ctime: time.Unix(int64(item.Ctime), 0),
		})
	}

	return backups, nil
}

func CreateBackup(userID, vmID uint64) (uint, error) {
	bkr, err := db.NewBackupRequest(BackupRequestTypeCreate, BackupRequestStatusPending, uint(vmID))
	if err != nil {
		logger.Error("failed to create backup request", "error", err)
		return 0, err
	}

	return bkr.ID, nil
}

func DeleteBackup(userID, vmID uint64, backupid string, since time.Time) (uint, error) {
	_, _, _, mcontent, err := listBackups(vmID, since)
	if err != nil {
		return 0, err
	}

	for _, item := range mcontent {
		h := hmac.New(sha256.New, nonce)
		h.Write([]byte(item.Volid))

		if hex.EncodeToString(h.Sum(nil)) == backupid && strings.Contains(item.Notes, BackupNoteString) {
			bkr, err := db.NewBackupRequest(BackupRequestTypeDelete, BackupRequestStatusPending, uint(vmID))
			if err != nil {
				logger.Error("failed to create backup request", "error", err)
				return 0, err
			}
			return bkr.ID, nil
		}
	}

	return 0, ErrBackupNotFound
}

func RestoreBackup(userID, vmID uint64, backupid string, since time.Time) (uint, error) {
	cluster, err := getProxmoxCluster(client)
	if err != nil {
		return 0, err
	}

	resources, err := getProxmoxResources(cluster, "vm")
	if err != nil {
		return 0, err
	}

	for _, r := range resources {
		if r.VMID == vmID {
			if r.Status != string(VMStatusStopped) {
				return 0, ErrInvalidVMState
			} else {
				break
			}
		}
	}

	bkr, err := db.NewBackupRequest(BackupRequestTypeRestore, BackupRequestStatusPending, uint(vmID))
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
