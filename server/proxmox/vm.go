package proxmox

import "samuelemusiani/sasso/server/db"

type VMStatus string

var (
	VMStatusRunning   VMStatus = "running"
	VMStatusStopped   VMStatus = "stopped"
	VMStatusSuspended VMStatus = "suspended"
	VMStatusUnknown   VMStatus = "unknown"
)

type VM struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

func GetVMsByUserID(userID uint) ([]VM, error) {
	db_vms, err := db.GetVMsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get VMs by user ID", "userID", userID, "error", err)
		return nil, err
	}

	vms := make([]VM, len(db_vms))

	for i := range vms {
		vms[i].ID = db_vms[i].ID
		// Status needs to be checked against the acctual Proxmox VM status
		vms[i].Status = string(vms[i].Status)
	}

	return vms, nil
}
