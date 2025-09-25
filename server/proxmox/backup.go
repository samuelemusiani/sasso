package proxmox

import (
	"context"
	"time"
)

type Backup struct {
	Volid string    `json:"volid"`
	VMID  uint64    `json:"vmid"`
	Ctime time.Time `json:"ctime"`
}

func ListBackups(vmID uint64, since time.Time) ([]Backup, error) {

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		logger.Error("failed to get proxmox cluster", "error", err)
		return nil, err
	}

	m, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.Error("failed to map VMID to Proxmox nodes", "error", err)
		return nil, err
	}

	nodeName, ok := m[vmID]
	if !ok {
		logger.Error("no Proxmox node found for VMID", "vmID", vmID)
		return nil, ErrVMNotFound
	}

	node, err := getProxmoxNode(client, nodeName)
	if err != nil {
		logger.Error("failed to get proxmox node", "error", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s, err := node.Storage(ctx, cBackup.Storage)
	defer cancel()
	if err != nil {
		logger.Error("failed to get storage info", "error", err)
		return nil, err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	mcontent, err := s.GetContent(ctx)
	defer cancel()
	if err != nil {
		logger.Error("failed to get storage content", "error", err)
		return nil, err
	}

	var backups []Backup
	for _, item := range mcontent {
		if item.VMID != vmID || time.Unix(int64(item.Ctime), 0).Before(since) {
			continue
		}
		backups = append(backups, Backup{
			Volid: item.Volid,
			VMID:  item.VMID,
			Ctime: time.Unix(int64(item.Ctime), 0),
		})
	}

	return backups, nil
}
