package proxmox

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"samuelemusiani/sasso/server/db"
)

type VMStatus string

var (
	VMStatusRunning   VMStatus = "running"
	VMStatusStopped   VMStatus = "stopped"
	VMStatusSuspended VMStatus = "suspended"
	VMStatusUnknown   VMStatus = "unknown"

	// The pre-status is before the main worker has acknowledged the creation or
	// deletion
	VMStatusPreCreating VMStatus = "pre-creating"
	VMStatusPreDeleting VMStatus = "pre-deleting"

	// This status is then the main worker has taken an action, but the vm
	// is not yet fully cloned or deleted.
	VMStatusCreating VMStatus = "creating"
	VMStatusDeleting VMStatus = "deleting"

	VMStatusPreConfiguring VMStatus = "pre-configuring"
	VMStatusConfiguring    VMStatus = "configuring"

	ErrVMNotFound error = errors.New("VM not found")
)

type VM struct {
	ID     uint64 `json:"id"`
	Status string `json:"status"`
	Cores  uint   `json:"cores"`
	RAM    uint   `json:"ram"`
	Disk   uint   `json:"disk"`
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
		vms[i].Cores = db_vms[i].Cores
		vms[i].RAM = db_vms[i].RAM
		vms[i].Disk = db_vms[i].Disk
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

func NewVM(userID uint, cores uint, ram uint, disk uint) (*VM, error) {
	user, err := db.GetUserByID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get user from database")
		return nil, err
	}

	vms, err := db.GetVMsByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get VMs by user ID")
		return nil, err
	}

	var currentCores uint = 0
	var currentRAM uint = 0
	var currentDisk uint = 0

	for _, vm := range vms {
		currentCores += vm.Cores
		currentRAM += vm.RAM
		currentDisk += vm.Disk
	}

	if currentCores+cores > user.MaxCores {
		return nil, ErrInsufficientResources
	}

	if currentRAM+ram > user.MaxRAM {
		return nil, ErrInsufficientResources
	}

	if currentDisk+disk > user.MaxDisk {
		return nil, ErrInsufficientResources
	}

	vmUserID, err := db.GetLastVMUserIDByUserID(userID)
	if err != nil {
		logger.With("userID", userID, "error", err).
			Error("Failed to get last VM user ID from database")
		return nil, err
	}

	vmUserID++ // Increment the VM user ID for the new VM
	VMID, err := generateFullVMID(userID, vmUserID)

	db_vm, err := db.NewVM(VMID, userID, vmUserID, string(VMStatusPreCreating), cores, ram, disk)
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

func DeleteVM(userID uint, vmID uint64) error {
	_, err := db.GetVMByUserIDAndVMID(userID, vmID)
	if err != nil {
		if err == db.ErrNotFound {
			logger.With("userID", userID, "vmID", vmID).
				Warn("VM not found for deletion")
			return ErrVMNotFound
		} else {
			logger.With("userID", userID, "vmID", vmID, "error", err).
				Error("Failed to get VM from database for deletion")
			return err
		}
	}

	if err := db.UpdateVMStatus(vmID, string(VMStatusPreDeleting)); err != nil {
		logger.With("userID", userID, "vmID", vmID, "error", err).
			Error("Failed to update VM status from database")
		return err
	}

	logger.With("userID", userID, "vmID", vmID).
		Info("VM set to 'deleting' successfully")

	return nil
}

func TestEndpointClone() {
	time.Sleep(5 * time.Second)
	first := true
	wasError := false

	for {
		if !isProxmoxReachable {
			time.Sleep(20 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		node, err := client.Node(ctx, cTemplate.Node)
		cancel() // Cancel immediately after the call

		if err != nil {
			logger.Error("Failed to get Proxmox node", "node", cTemplate.Node, "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		vm, err := node.VirtualMachine(ctx, cTemplate.VMID)
		cancel()
		if err != nil {
			logger.Error("Failed to get Proxmox VM", "vmid", cTemplate.VMID, "error", err)
			wasError = true
		} else if first {
			logger.Info("Proxmox VM is ready for cloning", "vmid", cTemplate.VMID, "status", vm.Status)
			first = false
		} else if wasError {
			logger.Info("Proxmox VM is back online for cloning", "vmid", cTemplate.VMID, "status", vm.Status)
			wasError = false
		}

		time.Sleep(10 * time.Second)
	}
}
