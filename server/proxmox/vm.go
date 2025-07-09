package proxmox

import (
	"fmt"
	"samuelemusiani/sasso/server/db"
	"strconv"
	"strings"
)

type VMStatus string

var (
	VMStatusRunning   VMStatus = "running"
	VMStatusStopped   VMStatus = "stopped"
	VMStatusSuspended VMStatus = "suspended"
	VMStatusUnknown   VMStatus = "unknown"
)

type VM struct {
	ID     uint64 `json:"id"`
	Status string `json:"status"`
}

func GetVMsByUserID(userID uint) ([]VM, error) {
	db_vms, err := db.GetVMsByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).
			Error("Failed to get VMs by user ID")
		return nil, err
	}

	vms := make([]VM, len(db_vms))

	for i := range vms {
		vms[i].ID = db_vms[i].ID
		// Status needs to be checked against the acctual Proxmox VM status
		vms[i].Status = string(db_vms[i].Status)
	}

	return vms, nil
}

// Generate a full VM ID based on the user ID and VM user ID.
func generateFullVMID(userID uint, vmUserID uint) (uint64, error) {
	svmid := fmt.Sprintf("%0*d%0*d", cClone.VMIDUserDigits, userID, cClone.VMIDVMDigits, vmUserID)

	svmid = strings.Replace(cClone.IDTemplate, "{{vmid}}", svmid, 1)

	if len(svmid) < 3 || len(svmid) > 9 {
		return 0, fmt.Errorf("invalid clone ID template length: %d", len(svmid))
	}

	vmid, err := strconv.ParseUint(svmid, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid clone ID template: %s", svmid)
	}

	return vmid, nil
}

func NewVM(userID uint) (*VM, error) {
	vmUserID, err := db.GetLastVMUserIDByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).
			Error("Failed to get last VM user ID from database")
		return nil, err
	}

	vmUserID++ // Increment the VM user ID for the new VM
	VMID, err := generateFullVMID(userID, vmUserID)

	db_vm, err := db.NewVM(VMID, userID, vmUserID, string(VMStatusUnknown))
	if err != nil {
		logger.With("userID", userID, "vmUserID", vmUserID, "error", err).
			Error("Failed to create new VM in database")
		return nil, err
	}

	vm := &VM{
		ID:     db_vm.ID,
		Status: string(db_vm.Status),
	}

	return vm, nil
}
