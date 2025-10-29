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

	vmMinCores uint = 1
	vmMinRAM   uint = 512 // in MB
)

type VM struct {
	ID                   uint64    `json:"id"`
	CreatedAt            time.Time `json:"-"`
	Status               string    `json:"status"`
	Name                 string    `json:"name"`
	Notes                string    `json:"notes"`
	Cores                uint      `json:"cores"`
	RAM                  uint      `json:"ram"`
	Disk                 uint      `json:"disk"`
	LifeTime             time.Time `json:"lifetime"`
	IncludeGlobalSSHKeys bool      `json:"include_global_ssh_keys"`
	OwnerID              uint      `json:"-"`
	OwnerType            string    `json:"-"`

	GroupID   uint   `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	// User role in the group (e.g., "member", "admin").
	// User is the one requesting the VM.
	GroupRole string `json:"group_role,omitempty"`
}

func convertDBVMToVM(db_vm *db.VM, groupID *uint, groupName, groupRole *string) *VM {
	vm := &VM{
		ID:                   db_vm.ID,
		CreatedAt:            db_vm.CreatedAt,
		Status:               string(db_vm.Status),
		Name:                 db_vm.Name,
		Notes:                db_vm.Notes,
		Cores:                db_vm.Cores,
		RAM:                  db_vm.RAM,
		Disk:                 db_vm.Disk,
		LifeTime:             db_vm.LifeTime,
		IncludeGlobalSSHKeys: db_vm.IncludeGlobalSSHKeys,
		OwnerID:              db_vm.OwnerID,
		OwnerType:            db_vm.OwnerType,
	}

	if groupID != nil && groupName != nil && groupRole != nil {
		vm.GroupID = *groupID
		vm.GroupName = *groupName
		vm.GroupRole = *groupRole
	}
	return vm
}

func GetVMsByUserID(userID uint) ([]VM, error) {
	db_vms, err := db.GetVMsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get VMs by user ID", "userID", userID, "error", err)
		return nil, err
	}

	vms := make([]VM, len(db_vms))

	for i := range vms {
		vms[i] = *convertDBVMToVM(&db_vms[i], nil, nil, nil)
	}

	groups, err := db.GetGroupsByUserID(userID)
	for _, g := range groups {
		gvms, err := db.GetVMsByGroupID(g.ID)
		if err != nil {
			logger.Error("Failed to get VMs by group ID", "groupID", g.ID, "error", err)
			return nil, err
		}
		role, err := db.GetUserRoleInGroup(userID, g.ID)
		if err != nil {
			logger.Error("Failed to get user role in group", "userID", userID, "groupID", g.ID, "error", err)
			return nil, err
		}

		for i := range gvms {
			vms = append(vms, *convertDBVMToVM(&gvms[i], &g.ID, &g.Name, &role))
		}
	}

	return vms, nil
}

func GetVMByID(VMID uint64, userID uint) (*VM, error) {
	db_vm, err := db.GetVMByID(VMID)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrVMNotFound
		}
		logger.Error("Failed to get VM by ID", "vmID", VMID, "error", err)
		return nil, err
	}

	var groupID *uint
	var groupName, role *string
	if db_vm.OwnerType == "Group" {
		group, err := db.GetGroupByID(db_vm.OwnerID)
		if err != nil {
			logger.Error("Failed to get group by ID for VM", "groupID", db_vm.OwnerID, "vmID", VMID, "error", err)
			return nil, err
		}
		r, err := db.GetUserRoleInGroup(userID, group.ID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				return nil, ErrVMNotFound
			}
			logger.Error("Failed to get user role in group for VM", "userID", userID, "groupID", group.ID, "vmID", VMID, "error", err)
			return nil, err
		}
		groupID = &group.ID
		groupName = &group.Name
		role = &r
	}

	return convertDBVMToVM(db_vm, groupID, groupName, role), nil
}

// Generate a full VM ID based on the user ID and VM user ID.
func generateFullVMID(group bool, ownerID uint, vmOwnerID uint) (uint64, error) {
	groupBit := 0
	if group {
		groupBit = 1
	}
	svmid := fmt.Sprintf("%d%0*d%0*d", groupBit, cClone.VMIDUserDigits, ownerID, cClone.VMIDVMDigits, vmOwnerID)

	svmid = strings.Replace(cClone.IDTemplate, "{{vmid}}", svmid, 1)

	if len(svmid) < 3 || len(svmid) > 9 {
		logger.Error("Invalid clone ID template length", "length", len(svmid), "vmid", svmid, "group", group, "ownerID", ownerID, "vmOwnerID", vmOwnerID)
		return 0, fmt.Errorf("invalid clone ID template length: %d", len(svmid))
	}

	vmid, err := strconv.ParseUint(svmid, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid clone ID template: %s", svmid)
	}

	return vmid, nil
}

func NewVM(userID uint, groupID *uint, name string, notes string, cores uint, ram uint, disk uint, lifeTime uint, includeGlobalSSHKeys bool) (*VM, error) {
	l := logger.With("userID", userID, "vmName", name)
	if groupID != nil {
		l = logger.With("groupID", *groupID)
	}

	if !vmNameRegex.MatchString(name) || len(name) > 16 {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("invalid name"))
	}

	if cores < vmMinCores {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("cores must be at least 1"))
	}
	if ram < vmMinRAM {
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

	var group *db.Group
	if groupID != nil {
		group, err = db.GetGroupByID(*groupID)
		if err != nil {
			logger.Error("Failed to get group from database", "groupID", *groupID, "error", err)
			return nil, err
		}
	}

	var exists bool
	if group != nil {
		exists, err = db.ExistsVMWithGroupIDAndName(*groupID, name)
	} else {
		exists, err = db.ExistsVMWithUserIDAndName(userID, name)
	}
	if err != nil {
		l.Error("Failed to check if VM name exists", "error", err)
		return nil, err
	} else if exists {
		return nil, errors.Join(ErrInvalidVMParam, errors.New("vm name already exists"))
	}

	var currentCores, currentRAM, currentDisk uint

	if group != nil {
		currentCores, currentRAM, currentDisk, err = db.GetVMResourcesByGroupID(*groupID)
	} else {
		currentCores, currentRAM, currentDisk, err = db.GetVMResourcesByUserID(userID)
	}

	if err != nil {
		l.Error("Failed to get current VM resources from database", "error", err)
		return nil, err
	}

	var maxCores, maxRAM, maxDisk uint
	if group != nil {
		maxCores, maxRAM, maxDisk, _, err = db.GetGroupResourceLimits(*groupID)
		if err != nil {
			l.Error("Failed to get group resource limits from database", "groupID", *groupID, "error", err)
			return nil, err
		}
	} else {
		maxCores, maxRAM, maxDisk = user.MaxCores, user.MaxRAM, user.MaxDisk
	}

	if currentCores+cores > maxCores {
		return nil, ErrInsufficientResources
	}

	if currentRAM+ram > maxRAM {
		return nil, ErrInsufficientResources
	}

	if currentDisk+disk > maxDisk {
		return nil, ErrInsufficientResources
	}

	var ids []uint
	if group != nil {
		ids, err = db.GetAllVMsIDsByGroupID(*groupID)
	} else {
		ids, err = db.GetAllVMsIDsByUserID(userID)
	}
	if err != nil {
		l.Error("Failed to get existing VM IDs from database", "error", err)
		return nil, err
	}

	uniqueOwnerID, err := getLastUsedUniqueOwnerIDInVMs(ids)
	if err != nil {
		l.Error("Failed to get last used unique owner ID in VMs", "error", err)
		return nil, err
	}

	uniqueOwnerID++ // Increment the VM user ID for the new VM
	ownerID := userID
	if group != nil {
		ownerID = *groupID
	}
	VMID, err := generateFullVMID(group != nil, ownerID, uniqueOwnerID)
	if err != nil {
		l.Error("Failed to generate full VM ID", "error", err)
		return nil, err
	}

	var db_vm *db.VM
	if group != nil {
		db_vm, err = db.NewVMForGroup(VMID, *groupID, string(VMStatusPreCreating), name, notes, cores, ram, disk, time.Now().AddDate(0, int(lifeTime), 0), includeGlobalSSHKeys)
	} else {
		db_vm, err = db.NewVMForUser(VMID, userID, string(VMStatusPreCreating), name, notes, cores, ram, disk, time.Now().AddDate(0, int(lifeTime), 0), includeGlobalSSHKeys)
	}

	if err != nil {
		l.Error("Failed to create new VM in database", "error", err)
		return nil, err
	}

	var groupName, role *string
	if group != nil {
		groupName = &group.Name
		r, err := db.GetUserRoleInGroup(userID, group.ID)
		if err != nil {
			l.Error("Failed to get user role in group for new VM", "groupID", group.ID, "error", err)
			return nil, err
		}
		role = &r
	}

	return convertDBVMToVM(db_vm, groupID, groupName, role), nil
}

func DeleteVM(group bool, ownerID, userID uint, vmID uint64) error {
	var err error
	var vm *db.VM
	if group {
		vm, err = db.GetVMByGroupIDAndVMID(ownerID, vmID)
	} else {
		vm, err = db.GetVMByUserIDAndVMID(userID, vmID)
	}
	if err != nil {
		if err == db.ErrNotFound {
			logger.Warn("VM not found for deletion", "userID", userID, "vmID", vmID)
			return ErrVMNotFound
		} else {
			logger.Error("Failed to get VM from database for deletion", "userID", userID, "vmID", vmID, "error", err)
			return err
		}
	}

	if group {
		role, err := db.GetUserRoleInGroup(userID, ownerID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				logger.Warn("User has no role in group for changing VM status", "userID", userID, "groupID", ownerID, "vmID", vmID)
				return ErrVMNotFound
			}
			logger.Error("Failed to get user role in group for changing VM status", "userID", userID, "groupID", ownerID, "vmID", vmID, "error", err)
			return err
		}
		if group && role != "admin" && role != "owner" {
			return ErrPermissionDenied
		}
	}

	vmStates := []string{string(VMStatusRunning), string(VMStatusStopped), string(VMStatusSuspended), string(VMStatusUnknown)}

	if !slices.Contains(vmStates, vm.Status) {
		logger.Warn("VM is not in a deletable state", "vmID", vmID, "status", vm.Status)
		return ErrInvalidVMState
	}

	return deleteVMBypass(vmID)
}

// deleteVMBypass deletes a VM directly without any checks. This is used by
// the main worker when processing VM deletions.
func deleteVMBypass(vmID uint64) error {
	if err := db.UpdateVMStatus(vmID, string(VMStatusPreDeleting)); err != nil {
		logger.Error("Failed to update VM status from database", "vmID", vmID, "error", err)
		return err
	}

	logger.Debug("VM set to 'deleting' successfully", "vmID", vmID)
	return nil
}

func changeVMStatusBypass(vmID uint64, action string) error {
	vm, err := db.GetVMByID(vmID)
	if err != nil {
		if err == db.ErrNotFound {
			logger.Warn("VM not found for changing status", "vmID", vmID)
			return ErrVMNotFound
		} else {
			logger.Error("Failed to get VM from database for changing status", "vmID", vmID, "error", err)
			return err
		}
	}

	switch action {
	case "start":
		if vm.Status != string(VMStatusStopped) {
			logger.Warn("VM is not in 'stopped' state, cannot start", "vmID", vmID, "status", vm.Status)
			return nil
		}
	case "stop", "restart":
		if vm.Status != string(VMStatusRunning) {
			logger.Warn("VM is not in 'running' state, cannot stop or restart", "vmID", vmID, "status", vm.Status)
			return nil
		}
	default:
		return ErrInvalidVMState
	}

	if (action == "start" || action == "restart") && vm.LifeTime.Before(time.Now()) {
		logger.Warn("VM lifetime has expired, cannot start or restart", "vmID", vmID, "lifetime", vm.LifeTime)
		return errors.Join(ErrInvalidVMState, errors.New("vm lifetime has expired; cannot start or restart"))
	}

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		logger.Error("Failed to get Proxmox cluster for changing VM status", "vmID", vmID, "error", err)
		return err
	}

	vmNodes, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.Error("Failed to map VM IDs to Proxmox nodes for changing VM status", "vmID", vmID, "error", err)
		return err
	}

	nodeName, exists := vmNodes[vmID]
	if !exists {
		logger.Error("VM ID not found in Proxmox cluster for changing VM status", "vmID", vmID)
		return ErrVMNotFound
	}

	node, err := getProxmoxNode(client, nodeName)
	if err != nil {
		logger.Error("Failed to get Proxmox node for changing VM status", "vmID", vmID, "node", nodeName, "error", err)
		return err
	}

	vmr, err := getProxmoxVM(node, int(vmID))
	if err != nil {
		logger.Error("Failed to get Proxmox VM for changing VM status", "vmID", vmID, "node", nodeName, "error", err)
		return ErrVMNotFound
	}

	switch action {
	case "start":
		if vmr.Status != "stopped" {
			logger.Warn("VM is not in 'stopped' state in Proxmox, cannot start", "vmID", vmID, "node", nodeName, "status", vmr.Status)
			return ErrInvalidVMState
		}
	case "stop", "restart":
		if vmr.Status != "running" {
			logger.Warn("VM is not in 'running' state in Proxmox, cannot stop or restart", "vmID", vmID, "node", nodeName, "status", vmr.Status)
			return ErrInvalidVMState
		}
	default:
		return ErrInvalidVMState
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		logger.Error(fmt.Sprintf("Failed to %s VM in Proxmox", action), "vmID", vmID, "node", nodeName, "error", err)
		return err
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to wait for Proxmox task completion when trying to %s VM", action), "vmID", vmID, "node", nodeName, "error", err)
		return err
	}

	if !isSuccessful {
		logger.Error("Proxmox task to start VM was not successful", "vmID", vmID, "node", nodeName)
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
		logger.Error("Failed to update VM status from database", "vmID", vmID, "error", err)
		return err
	}

	logger.Debug(fmt.Sprintf("VM %sed successfully", action), "vmID", vmID)
	return nil
}

// If the VM belongs to a user, userID and ownerID are the same.
func ChangeVMStatus(group bool, ownerID, userID uint, vmID uint64, action string) error {
	var err error
	var vm *db.VM
	if group {
		vm, err = db.GetVMByGroupIDAndVMID(ownerID, vmID)
	} else {
		vm, err = db.GetVMByUserIDAndVMID(ownerID, vmID)
	}
	if err != nil {
		if err == db.ErrNotFound {
			logger.Warn("VM not found for changing status", "ownerID", ownerID, "vmID", vmID, "group", group)
			return ErrVMNotFound
		} else {
			logger.Error("Failed to get VM from database for changing status", "userID", userID, "vmID", vmID, "error", err)
			return err
		}
	}

	if group {
		role, err := db.GetUserRoleInGroup(userID, ownerID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				logger.Warn("User has no role in group for changing VM status", "userID", userID, "groupID", ownerID, "vmID", vmID)
				return ErrVMNotFound
			}
			logger.Error("Failed to get user role in group for changing VM status", "userID", userID, "groupID", ownerID, "vmID", vmID, "error", err)
			return err
		}
		if group && role != "admin" && role != "owner" {
			return ErrPermissionDenied
		}
	}

	vmStates := []string{string(VMStatusRunning), string(VMStatusStopped), string(VMStatusSuspended)}
	if !slices.Contains(vmStates, vm.Status) {
		logger.Warn("VM is not in a valid state for changing status", "vmID", vmID, "status", vm.Status)
		return ErrInvalidVMState
	}

	return changeVMStatusBypass(vmID, action)
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

// VMs for a single user or group have a unique ID that is incremented. This
// is used to generate the full VM ID. This function takes all the existing
// VM IDs for a user or group and returns the highest unique owner ID used.
// We do this here because it's based on the template and the DB should
// not need to know about that.
func getLastUsedUniqueOwnerIDInVMs(ids []uint) (uint, error) {
	// like 600{{vmid}} or 60{{vmid}}0
	first := strings.Index("60{{vmid}}", "{{vmid}}") //3
	if first == -1 {
		panic("invalid clone ID template")
	}

	var maxID uint = 0
	for _, id := range ids {
		sid := strconv.Itoa(int(id))
		if len(sid) < 1+cClone.VMIDUserDigits {
			logger.Error("VM ID in database is shorter than expected", "id", id)
			continue
		}
		sUniqueID := sid[first+1+cClone.VMIDUserDigits:]
		uniqueOwnerID, err := strconv.Atoi(sUniqueID)
		if err != nil {
			logger.Error("Failed to convert unique owner ID to integer", "id", sid, "error", err)
			continue
		}

		if uint(uniqueOwnerID) > maxID {
			maxID = uint(uniqueOwnerID)
		}
	}
	return maxID, nil
}

// Like above (getLastUsedUniqueOwnerIDInVMs), but for a single VM ID.
func getUniqueOwnerIDInVM(id uint) (uint, error) {
	// like 600{{vmid}} or 60{{vmid}}0
	first := strings.Index(cClone.IDTemplate, "{{vmid}}")
	if first == -1 {
		panic("invalid clone ID template")
	}

	sid := strconv.Itoa(int(id))
	if len(sid) < 1+cClone.VMIDUserDigits {
		return 0, fmt.Errorf("invalid VM ID in database: %d", id)
	}

	sUniqueID := sid[first+1+cClone.VMIDUserDigits:]
	uniqueOwnerID, err := strconv.Atoi(sUniqueID)
	if err != nil {
		return 0, fmt.Errorf("failed to convert unique owner ID to integer: %v", err)
	}
	return uint(uniqueOwnerID), nil
}

func UpdateVMResources(VMID uint64, cores, ram, disk uint) error {
	vm, err := db.GetVMByID(VMID)
	if err != nil {
		logger.Error("Failed to get VM from database for updating resources", "vmID", VMID, "error", err)
		return err
	}

	if cores < vmMinCores {
		return errors.Join(ErrInvalidVMParam, errors.New("cores must be at least 1"))
	}

	if ram < vmMinRAM {
		return errors.Join(ErrInvalidVMParam, errors.New("ram must be at least 512 MB"))
	}

	if vm.Disk > disk {
		return errors.Join(ErrInvalidVMParam, errors.New("disk size can only be increased"))
	}

	vmStates := []string{string(VMStatusRunning), string(VMStatusStopped), string(VMStatusSuspended), string(VMStatusUnknown)}

	if !slices.Contains(vmStates, vm.Status) {
		logger.Warn("VM is not in a state for resource updates", "vmID", VMID, "status", vm.Status)
		return ErrInvalidVMState
	}

	var group *db.Group = nil
	if vm.OwnerType == "Group" {
		group, err = db.GetGroupByID(vm.OwnerID)
		if err != nil {
			logger.Error("Failed to get group from database for updating VM resources", "groupID", vm.OwnerID, "vmID", VMID, "error", err)
			return err
		}
	}

	var currentCores, currentRAM, currentDisk uint

	if group != nil {
		currentCores, currentRAM, currentDisk, err = db.GetVMResourcesByGroupID(group.ID)
	} else {
		currentCores, currentRAM, currentDisk, err = db.GetVMResourcesByUserID(vm.OwnerID)
	}

	var maxCores, maxRAM, maxDisk uint
	if group != nil {
		maxCores, maxRAM, maxDisk, _, err = db.GetGroupResourceLimits(group.ID)
		if err != nil {
			logger.Error("Failed to get group resource limits from database", "groupID", group.ID, "error", err)
			return err
		}
	} else {
		user, err := db.GetUserByID(vm.OwnerID)
		if err != nil {
			logger.Error("Failed to get user from database for updating VM resources", "userID", vm.OwnerID, "vmID", VMID, "error", err)
			return err
		}
		maxCores, maxRAM, maxDisk = user.MaxCores, user.MaxRAM, user.MaxDisk
	}

	if currentCores-uint(vm.Cores)+cores > maxCores ||
		currentRAM-uint(vm.RAM)+ram > maxRAM ||
		currentDisk-uint(vm.Disk)+disk > maxDisk {
		return ErrInsufficientResources
	}

	err = db.UpdateVMResources(VMID, cores, ram, disk)
	if err != nil {
		logger.Error("Failed to update VM resources in database", "vmID", VMID, "error", err)
		return err
	}

	err = db.UpdateVMStatus(VMID, string(VMStatusPreConfiguring))
	if err != nil {
		logger.Error("Failed to update VM status to pre-configuring in database", "vmID", VMID, "error", err)
		return err
	}

	return nil
}
