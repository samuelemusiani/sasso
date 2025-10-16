package proxmox

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"samuelemusiani/sasso/server/db"

	goprox "github.com/luthermonson/go-proxmox"
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

	VMCloneDiskSizeGB uint = 4 // Minimum disk size in GB for a VM clone

	ErrVMNotFound     error = errors.New("VM not found")
	ErrInvalidVMState error = errors.New("invalid VM state for this action")
	ErrInvalidVMParam error = errors.New("invalid VM parameter")

	vmNameRegex = regexp.MustCompile(`^\w+(\w|-)*\w+$`)
	vmLifeTimes = []uint{1, 3, 6, 12}
)

type VM struct {
	ID                   uint64    `json:"id"`
	UserID               uint      `json:"user_id"`
	Status               string    `json:"status"`
	Name                 string    `json:"name"`
	Notes                string    `json:"notes"`
	Cores                uint      `json:"cores"`
	RAM                  uint      `json:"ram"`
	Disk                 uint      `json:"disk"`
	LifeTime             time.Time `json:"lifetime"`
	IncludeGlobalSSHKeys bool      `json:"include_global_ssh_keys"`
	CreatedAt            time.Time `json:"-"`
}

func convertDBVMToVM(db_vm *db.VM) *VM {
	return &VM{
		ID:                   db_vm.ID,
		Status:               string(db_vm.Status),
		Name:                 db_vm.Name,
		Notes:                db_vm.Notes,
		Cores:                db_vm.Cores,
		RAM:                  db_vm.RAM,
		Disk:                 db_vm.Disk,
		LifeTime:             db_vm.LifeTime,
		UserID:               db_vm.UserID,
		IncludeGlobalSSHKeys: db_vm.IncludeGlobalSSHKeys,
		CreatedAt:            db_vm.CreatedAt,
	}
}

func GetVMsByUserID(userID uint) ([]VM, error) {
	db_vms, err := db.GetVMsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get VMs by user ID", "userID", userID, "error", err)
		return nil, err
	}

	vms := make([]VM, len(db_vms))

	for i := range vms {
		vms[i] = *convertDBVMToVM(&db_vms[i])
	}

	return vms, nil
}

func GetVMByID(VMID uint64) (*VM, error) {
	db_vm, err := db.GetVMByID(VMID)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrVMNotFound
		}
		logger.Error("Failed to get VM by ID", "vmID", VMID, "error", err)
		return nil, err
	}

	return convertDBVMToVM(db_vm), nil
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

func NewVM(userID uint, name string, notes string, cores uint, ram uint, disk uint, lifeTime uint, includeGlobalSSHKeys bool) (*VM, error) {

	if !vmNameRegex.MatchString(name) || len(name) > 16 {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("invalid name"))
	}

	if cores < 1 {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("cores must be at least 1"))
	}
	if ram < 512 {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("ram must be at least 512 MB"))
	}
	if disk < VMCloneDiskSizeGB {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("disk must be at least 4 GB"))
	}

	if !slices.Contains(vmLifeTimes, lifeTime) {
		err := fmt.Errorf("lifetime must be one of the following values: %v", vmLifeTimes)
		return nil, errors.Join(ErrInvalidVMParam, err)
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		logger.Error("Failed to get user from database", "userID", userID, "error", err)
		return nil, err
	}

	exists, err := db.ExistsVMWithUserIdAndName(userID, name)
	if err != nil {
		logger.Error("Failed to check if VM name exists", "userID", userID, "name", name, "error", err)
		return nil, err
	}
	if exists {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("vm name already exists"))
	}

	currentCores, currentRAM, currentDisk, err := db.GetVMResourcesByUserID(userID)
	if err != nil {
		logger.Error("Failed to get current VM resources from database", "userID", userID, "error", err)
		return nil, err
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
		logger.Error("Failed to get last VM user ID from database", "userID", userID, "error", err)
		return nil, err
	}

	vmUserID++ // Increment the VM user ID for the new VM
	VMID, err := generateFullVMID(userID, vmUserID)

	db_vm, err := db.NewVM(VMID, userID, vmUserID, string(VMStatusPreCreating), name, notes, cores, ram, disk, time.Now().AddDate(0, int(lifeTime), 0), includeGlobalSSHKeys)
	if err != nil {
		logger.Error("Failed to create new VM in database", "userID", userID, "vmUserID", vmUserID, "error", err)
		return nil, err
	}

	return convertDBVMToVM(db_vm), nil
}

func DeleteVM(userID uint, vmID uint64) error {
	_, err := db.GetVMByUserIDAndVMID(userID, vmID)
	if err != nil {
		if err == db.ErrNotFound {
			logger.Warn("VM not found for deletion", "userID", userID, "vmID", vmID)
			return ErrVMNotFound
		} else {
			logger.Error("Failed to get VM from database for deletion", "userID", userID, "vmID", vmID, "error", err)
			return err
		}
	}

	if err := db.UpdateVMStatus(vmID, string(VMStatusPreDeleting)); err != nil {
		logger.Error("Failed to update VM status from database", "userID", userID, "vmID", vmID, "error", err)
		return err
	}

	logger.Debug("VM set to 'deleting' successfully", "userID", userID, "vmID", vmID)

	return nil
}

func ChangeVMStatus(userID uint, vmID uint64, action string) error {
	vm, err := db.GetVMByUserIDAndVMID(userID, vmID)
	if err != nil {
		if err == db.ErrNotFound {
			logger.Warn("VM not found for changing status", "userID", userID, "vmID", vmID)
			return ErrVMNotFound
		} else {
			logger.Error("Failed to get VM from database for changing status", "userID", userID, "vmID", vmID, "error", err)
			return err
		}
	}

	switch action {
	case "start":
		if vm.Status != string(VMStatusStopped) {
			logger.Warn("VM is not in 'stopped' state, cannot start", "userID", userID, "vmID", vmID, "status", vm.Status)
			return nil
		}
	case "stop", "restart":
		if vm.Status != string(VMStatusRunning) {
			logger.Warn("VM is not in 'running' state, cannot stop or restart", "userID", userID, "vmID", vmID, "status", vm.Status)
			return nil
		}
	default:
		return ErrInvalidVMState
	}

	if (action == "start" || action == "restart") && vm.LifeTime.Before(time.Now()) {
		logger.Warn("VM lifetime has expired, cannot start or restart", "userID", userID, "vmID", vmID, "lifetime", vm.LifeTime)
		return errors.Join(ErrInvalidVMState, errors.New("vm lifetime has expired; cannot start or restart"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	clustr, err := client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox cluster for changing VM status", "userID", userID, "vmID", vmID, "error", err)
	}

	vmNodes, err := mapVMIDToProxmoxNodes(clustr)
	if err != nil {
		logger.Error("Failed to map VM IDs to Proxmox nodes for changing VM status", "userID", userID, "vmID", vmID, "error", err)
	}

	nodeName, exists := vmNodes[vmID]
	if !exists {
		logger.Error("VM ID not found in Proxmox cluster for changing VM status", "userID", userID, "vmID", vmID)
		return ErrVMNotFound
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	node, err := client.Node(ctx, nodeName)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox node for changing VM status", "userID", userID, "vmID", vmID, "node", nodeName, "error", err)
		return err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	vmr, err := node.VirtualMachine(ctx, int(vmID))
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox VM for changing VM status", "userID", userID, "vmID", vmID, "node", nodeName, "error", err)
		return ErrVMNotFound
	}

	switch action {
	case "start":
		if vmr.Status != "stopped" {
			logger.Warn("VM is not in 'stopped' state in Proxmox, cannot start", "userID", userID, "vmID", vmID, "node", nodeName, "status", vmr.Status)
			return ErrInvalidVMState
		}
	case "stop", "restart":
		if vmr.Status != "running" {
			logger.Warn("VM is not in 'running' state in Proxmox, cannot stop or restart", "userID", userID, "vmID", vmID, "node", nodeName, "status", vmr.Status)
			return ErrInvalidVMState
		}
	default:
		return ErrInvalidVMState
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	var task *goprox.Task
	switch action {
	case "start":
		task, err = vmr.Start(ctx)
	case "stop":
		task, err = vmr.Stop(ctx)
	case "restart":
		task, err = vmr.Reset(ctx)
	}
	cancel()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to %s VM in Proxmox", action), "userID", userID, "vmID", vmID, "node", nodeName, "error", err)
		return err
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to wait for Proxmox task completion when trying to %s VM", action), "userID", userID, "vmID", vmID, "node", nodeName, "error", err)
		return err
	}

	if !isSuccessful {
		logger.Error("Proxmox task to start VM was not successful", "userID", userID, "vmID", vmID, "node", nodeName)
		return ErrTaskFailed
	}

	var vmStatus VMStatus
	switch action {
	case "start":
		vmStatus = VMStatusRunning
	case "stop":
		vmStatus = VMStatusStopped
	case "restart":
		vmStatus = VMStatusRunning
	}

	if err := db.UpdateVMStatus(vmID, string(vmStatus)); err != nil {
		logger.Error("Failed to update VM status from database", "userID", userID, "vmID", vmID, "error", err)
		return err
	}

	logger.Debug(fmt.Sprintf("VM %sed successfully", action), "userID", userID, "vmID", vmID)

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

		node, err := getProxmoxNode(client, cTemplate.Node)
		if err != nil {
			logger.Error("Failed to get Proxmox node", "node", cTemplate.Node, "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		vm, err := getProxmoxVM(node, cTemplate.VMID)
		if err != nil {
			logger.Error("Failed to get Proxmox VM", "vmid", cTemplate.VMID, "error", err)
			wasError = true
		} else if first {
			logger.Info("Proxmox VM is ready for cloning", "vmid", cTemplate.VMID, "status", vm.Status)
			first = false

			s, ok := vm.VirtualMachineConfig.SCSIs["scsi0"]
			if ok {
				sto, err := parseStorageFromString(s)
				if err != nil {
					logger.Error("Failed to parse storage from VM config", "vmid", cTemplate.VMID, "error", err)
				} else {
					VMCloneDiskSizeGB = sto.Size
				}
			}

		} else if wasError {
			logger.Info("Proxmox VM is back online for cloning", "vmid", cTemplate.VMID, "status", vm.Status)
			wasError = false
		}

		time.Sleep(10 * time.Second)
	}
}

func UpdateVMLifetime(VMID uint64, extendBy uint) error {
	vm, err := db.GetVMByID(VMID)
	if err != nil {
		logger.Error("Failed to get VM from database for updating lifetime", "vmID", VMID, "error", err)
		return err
	}

	if extendBy == 0 || extendBy > 3 {
		return errors.Join(ErrInvalidVMParam, errors.New("extend_by must be 1, 2 or 3"))
	}

	months := int(extendBy / 2)
	days := int((extendBy % 2) * 15)
	if vm.LifeTime.After(time.Now().AddDate(0, months, days)) {
		return errors.Join(ErrInvalidVMParam, errors.New("cannot update lifetime. Too soon"))
	}

	err = db.UpdateVMLifetime(VMID, vm.LifeTime.AddDate(0, int(extendBy), 0))
	if err != nil {
		logger.Error("Failed to update VM lifetime in database", "vmID", VMID, "error", err)
		return err
	}
	return nil
}
