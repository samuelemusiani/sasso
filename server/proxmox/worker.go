// Sasso will write to the DB what the current state should look like. The
// worker will read the DB and take take care of all the operations that needs
// to be done

package proxmox

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/notify"

	gprox "github.com/luthermonson/go-proxmox"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type stringTime struct {
	Value string
	Time  time.Time
}

var (
	vmStatusTimeMap map[uint64]stringTime = make(map[uint64]stringTime)

	lastConfigureSSHKeysTime time.Time = time.Time{}

	workerContext    context.Context    = nil
	workerCancelFunc context.CancelFunc = nil
	workerReturnChan chan error         = make(chan error, 1)
)

func StartWorker() {
	workerContext, workerCancelFunc = context.WithCancel(context.Background())
	go func() {
		workerReturnChan <- worker(workerContext)
		close(workerReturnChan)
	}()
}

func ShutdownWorker() error {
	if workerCancelFunc != nil {
		workerCancelFunc()
	}
	var err error = nil
	if workerReturnChan != nil {
		err = <-workerReturnChan
	}
	if err != nil && err != context.Canceled {
		return err
	} else {
		return nil
	}
}

func worker(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		// Just a small delay to let other components start
	}

	logger.Info("Proxmox worker started")

	timeToWait := 10 * time.Second

	for {
		// Handle graceful shutdown at the start of each cycle
		select {
		case <-ctx.Done():
			logger.Info("Proxmox worker shutting down")
			return ctx.Err()
		case <-time.After(timeToWait):
		}

		now := time.Now()

		// For all VMs we must check the status and take the necessary actions
		if !isProxmoxReachable {
			time.Sleep(20 * time.Second)
			continue
		}

		objectCountHelper()

		cluster, err := getProxmoxCluster(client)
		if err != nil {
			logger.Error("Failed to get Proxmox cluster", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		workerCycleDurationObserve("create_vnets", func() { createVNets(cluster) })
		workerCycleDurationObserve("delete_vnets", func() { deleteVNets(cluster) })
		workerCycleDurationObserve("update_vnets", func() { updateVNets(cluster) })

		workerCycleDurationObserve("create_vms", func() { createVMs() })
		workerCycleDurationObserve("update_vms", func() { updateVMs(cluster) })

		workerCycleDurationObserve("lifetime_vms", func() { enforceVMLifetimes() })

		vmNodes, err := mapVMIDToProxmoxNodes(cluster)
		if err != nil {
			logger.Error("Failed to map VMID to Proxmox nodes", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		workerCycleDurationObserve("delete_vms", func() { deleteVMs(vmNodes) })
		workerCycleDurationObserve("configure_ssh_keys", func() { configureSSHKeys(vmNodes) })
		workerCycleDurationObserve("configure_vms", func() { configureVMs(vmNodes) })

		workerCycleDurationObserve("create_interfaces", func() { createInterfaces(vmNodes) })
		workerCycleDurationObserve("delete_interfaces", func() { deleteInterfaces(vmNodes) })

		workerCycleDurationObserve("delete_backups", func() { deleteBackups(vmNodes) })
		workerCycleDurationObserve("restore_backups", func() { restoreBackups(vmNodes) })
		workerCycleDurationObserve("create_backups", func() { createBackups(vmNodes) })

		elapsed := time.Since(now)
		workerCycleDuration.Observe(elapsed.Seconds())
		if elapsed < 10*time.Second {
			timeToWait = 10*time.Second - elapsed
		} else {
			timeToWait = 0
		}
	}
}

func objectCountHelper() {
	vmsCount, err := db.CountVMs()
	if err != nil {
		logger.Error("Failed to count VMs in DB", "error", err)
	} else {
		objectCountSet("vms", vmsCount)
	}

	interfacesCount, err := db.CountInterfaces()
	if err != nil {
		logger.Error("Failed to count interfaces in DB", "error", err)
	} else {
		objectCountSet("interfaces", interfacesCount)
	}

	netsCount, err := db.CountVNets()
	if err != nil {
		logger.Error("Failed to count VNets in DB", "error", err)
	} else {
		objectCountSet("vnets", netsCount)
	}

	countPortFowards, err := db.CountPortForwards()
	if err != nil {
		logger.Error("Failed to count port forwards in DB", "error", err)
	} else {
		objectCountSet("port_forwards", countPortFowards)
	}
}

func createVNets(cluster *gprox.Cluster) {
	logger.Debug("Creating VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VMStatusPreCreating))
	if err != nil {
		logger.Error("Failed to get VNets with 'pre-creating' status", "error", err)
		return
	}

	if len(vnets) == 0 {
		return
	}

	for _, v := range vnets {
		options := &gprox.VNetOptions{
			Name:      v.Name,
			Zone:      cNetwork.SDNZone,
			Tag:       v.Tag,
			VlanAware: v.VlanAware,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := cluster.NewSDNVNet(ctx, options)
		cancel()
		if err != nil {
			logger.Error("Failed to create VNet in Proxmox", "vnet", v.Name, "error", err)
			continue
		}

		err = db.UpdateVNetStatus(v.ID, string(VNetStatusCreating))
		if err != nil {
			logger.Error("Failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusCreating, "err", err)
			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)
	cancel()
	if err != nil {
		logger.Error("Failed to apply SDN changes in Proxmox", "error", err)
		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if err != nil {
		logger.Error("Failed to wait for Proxmox task completion", "error", err)
		return
	}

	if !isSuccessful {
		logger.Error("Failed to apply SDN changes in Proxmox")
		// Set all VNets status to 'unknown'
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("Failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}
		}
		return
	} else {
		logger.Debug("SDN changes applied successfully")
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusReady))
			if err != nil {
				logger.Error("Failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusReady, "err", err)
			}
		}
	}
}

func deleteVNets(cluster *gprox.Cluster) {
	logger.Debug("Deleting VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VNetStatusPreDeleting))
	if err != nil {
		logger.Error("Failed to get VNets with 'pre-deleting' status", "error", err)
		return
	}

	if len(vnets) == 0 {
		return
	}

	for _, v := range vnets {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := cluster.DeleteSDNVNet(ctx, v.Name)
		cancel()
		if err != nil {
			logger.Error("Failed to delete VNet from Proxmox", "vnet", v.Name, "error", err)
			continue
		}

		err = db.UpdateVNetStatus(v.ID, string(VNetStatusDeleting))
		if err != nil {
			logger.Error("Failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusDeleting, "err", err)
			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)
	cancel()
	if err != nil {
		logger.Error("Failed to apply SDN changes in Proxmox", "error", err)
		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if isSuccessful {
		logger.Debug("SDN changes applied successfully")
		for _, v := range vnets {
			err = db.DeleteNetByID(v.ID)
			if err != nil {
				logger.Error("Failed to delete VNet from DB", "vnet", v.Name, "err", err)
			}
		}
	} else {
		logger.Error("Failed to apply SDN changes in Proxmox")
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("Failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}
		}
	}
}

// createVMs creates VMs from proxmox that are in the 'pre-creating' status.
func createVMs() {
	logger.Debug("Creating VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreCreating))
	if err != nil {
		logger.Error("Failed to get VMs with 'creating' status", "error", err)
		return
	}

	if len(vms) == 0 {
		return
	}

	node, err := getProxmoxNode(client, cTemplate.Node)
	if err != nil {
		return
	}

	templateVm, err := getProxmoxVM(node, cTemplate.VMID)
	if err != nil {
		return
	}

	// https://github.com/luthermonson/go-proxmox/issues/102
	var optionFull uint8
	if cClone.Full {
		optionFull = 1
	} else {
		optionFull = 0
	}

	for _, v := range vms {
		if v.Status != string(VMStatusPreCreating) {
			continue
		}
		logger.Debug("Cloning VM", "vmid", v.ID)

		var vmName string
		if cClone.UserVMNames {
			vmName = v.Name
		} else {
			s := "sasso-%0" + strconv.Itoa(cClone.VMIDVMDigits) + "d"
			uniqueID, err := getUniqueOwnerIDInVM(uint(v.ID))
			if err != nil {
				logger.Error("Failed to get unique owner ID for VM naming", "vmid", v.ID, "err", err)
				continue
			}
			vmName = fmt.Sprintf(s, uniqueID)
		}

		cloningOptions := gprox.VirtualMachineCloneOptions{
			Full:   optionFull,
			Target: cClone.TargetNode,
			Name:   vmName,
		}

		// Create the VM in Proxmox
		// Creation implies cloning a template
		cloningOptions.NewID = int(v.ID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, task, err := templateVm.Clone(ctx, &cloningOptions)
		cancel()
		if err != nil {
			logger.Error("Failed to clone VM", "vmid", v.ID, "error", err)
			continue
		}
		err = db.UpdateVMStatus(v.ID, string(VMStatusCreating))
		if err != nil {
			logger.Error("Failed to update status of VM", "vmid", v.ID, "new_status", VMStatusCreating, "err", err)
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(task)
		if isSuccessful {
			err = db.UpdateVMStatus(v.ID, string(VMStatusPreConfiguring))
			if err != nil {
				logger.Error("Failed to update status of VM", "vmid", v.ID, "new_status", VMStatusStopped, "err", err)
			}
		} else {
			// We could set the status as pre-creating to trigger a recreation, but
			// for now we just set it to unknown
			err = db.UpdateVMStatus(v.ID, string(VMStatusUnknown))
			if err != nil {
				logger.Error("Failed to update status of VM", "vmid", v.ID, "new_status", VMStatusUnknown, "err", err)
			}
		}
	}
}

// deleteVMs deletes VMs from proxmox that are in the 'pre-deleting' status.
func deleteVMs(VMLocation map[uint64]string) {
	logger.Debug("Deleting VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreDeleting))
	if err != nil {
		logger.Error("Failed to get VMs with 'deleting' status", "error", err)
		return
	}

	for _, v := range vms {
		logger.Debug("Deleting VM", "vmid", v.ID)

		err := db.DeleteAllInterfacesByVMID(v.ID)
		if err != nil {
			logger.Error("Failed to delete interfaces for VM", "vmid", v.ID, "err", err)
		}

		nodeName, ok := VMLocation[v.ID]
		if !ok {
			logger.Error("Can't delete VM. Not found on cluster resources", "vmid", v.ID)

			// If the VM is not found on Proxmox, we just delete it from the DB
			err = db.DeleteVMByID(v.ID)
			if err != nil {
				logger.Error("Failed to delete VM", "vmid", v.ID, "err", err)
			}
			continue
		}

		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			continue
		}

		vm, err := getProxmoxVM(node, int(v.ID))
		if err != nil {
			continue
		}

		if vm.Status == "running" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			task, err := vm.Stop(ctx)
			cancel()
			if err != nil {
				logger.Error("Can't stop VM before deletion", "err", err, "vmid", v.ID)
				continue
			}
			isSuccessful, err := waitForProxmoxTaskCompletion(task)
			if err != nil {
				logger.Error("Can't wait for stop VM task completion", "err", err, "vmid", v.ID)
				continue
			}
			if !isSuccessful {
				logger.Error("Can't stop VM before deletion", "vmid", v.ID)
				continue
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		task, err := vm.Delete(ctx)
		cancel()
		if err != nil {
			logger.Error("Can't delete VM", "err", err, "vmid", v.ID)
			continue
		}
		err = db.UpdateVMStatus(v.ID, string(VMStatusDeleting))
		if err != nil {
			logger.Warn("Failed to update status of VM", "vmid", v.ID, "new_status", VMStatusDeleting, "err", err)
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(task)
		cancel()
		if isSuccessful {
			err = db.DeleteVMByID(v.ID)
			if err != nil {
				logger.Error("Failed to delete VM", "vmid", v.ID, "err", err)
			}
		} else {
			// We could set the status as pre-creating to trigger a recreation, but
			// for now we just set it to unknown
			err = db.UpdateVMStatus(v.ID, string(VMStatusUnknown))
			if err != nil {
				logger.Error("Failed to update status of VM", "vmid", v.ID, "new_status", VMStatusUnknown, "err", err)
			}
		}
	}
}

func updateVNets(cluster *gprox.Cluster) {
	logger.Debug("Updating VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VNetStatusReconfiguring))
	if err != nil {
		logger.Error("Failed to get VNets with status", "status", VNetStatusReconfiguring, "error", err)
		return
	}

	if len(vnets) == 0 {
		return
	}

	for _, v := range vnets {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		vnet, err := cluster.SDNVNet(ctx, v.Name)
		cancel()
		if err != nil {
			logger.Error("Failed to get VNet from Proxmox", "vnet", v.Name, "error", err)
			continue
		}

		if v.VlanAware {
			vnet.VlanAware = 1
		} else {
			vnet.VlanAware = 0
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		err = cluster.UpdateSDNVNet(ctx, vnet)
		cancel()
		if err != nil {
			logger.Error("Failed to update VNet in Proxmox", "vnet", v.Name, "error", err)
			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)
	cancel()
	if err != nil {
		logger.Error("Failed to apply SDN changes in Proxmox", "error", err)
		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if isSuccessful {
		logger.Debug("SDN changes applied successfully")
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusReady))
			if err != nil {
				logger.Error("Failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusReady, "err", err)
			}
		}
	} else {
		logger.Error("Failed to apply SDN changes in Proxmox")
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("Failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}
		}
	}
}

// This function configures VMs that are in the 'pre-configuring' status.
// Configuration includes setting the number of cores, RAM and disk size
func configureVMs(vmNodes map[uint64]string) {
	logger.Debug("Configuring VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreConfiguring))
	if err != nil {
		logger.Error("Failed to get VMs with 'pre-configuring' status", "error", err)
		return
	}

	for _, v := range vms {
		nodeName, ok := vmNodes[v.ID]
		if !ok {
			logger.Error("Can't configure VM. Not found on cluster resources", "vmid", v.ID)
			continue
		}
		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			continue
		}

		vm, err := getProxmoxVM(node, int(v.ID))
		if err != nil {
			continue
		}
		logger.Debug("Configuring VM", "vmid", v.ID)

		if vm.VirtualMachineConfig.Cores != int(v.Cores) {
			coresOption := gprox.VirtualMachineOption{
				Name:  "cores",
				Value: v.Cores,
			}
			isSuccessful, err := configureVM(vm, coresOption)
			if err != nil {
				logger.Error("Failed to set cores on VM", "vmid", v.ID, "err", err)
				return
			}
			logger.Debug("Task finished", "isSuccessful", isSuccessful)
			if !isSuccessful {
				logger.Error("Failed to set cores on VM", "vmid", v.ID)
			}
		}

		if uint(vm.VirtualMachineConfig.Memory) != v.RAM {
			ramOption := gprox.VirtualMachineOption{
				Name:  "memory",
				Value: v.RAM,
			}
			isSuccessful, err := configureVM(vm, ramOption)
			if err != nil {
				logger.Error("Failed to set ram on VM", "vmid", v.ID, "err", err)
				continue
			}
			logger.Debug("Task finished", "isSuccessful", isSuccessful)
			if !isSuccessful {
				logger.Error("Failed to set ram on VM", "vmid", v.ID)
			}
		}

		scsi0, ok := vm.VirtualMachineConfig.SCSIs["scsi0"]
		if !ok {
			logger.Error("Failed to find SCSI0 on VM", "vmid", v.ID)
			continue
		}
		st, err := parseStorageFromString(scsi0)
		if err != nil {
			logger.Error("Failed to parse storage on SCSI0", "vmid", v.ID, "scsi0", scsi0)
			continue
		}

		if st.Size < uint(v.Disk) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			diff := uint(v.Disk) - st.Size
			t, err := vm.ResizeDisk(ctx, "scsi0", fmt.Sprintf("+%dG", diff))
			cancel()
			if err != nil {
				logger.Error("Failed to set resize disk on VM", "vmid", v.ID, "err", err)
				continue
			}

			isSuccessful, err := waitForProxmoxTaskCompletion(t)
			logger.Debug("Task finished", "isSuccessful", isSuccessful)
			if !isSuccessful {
				logger.Error("Failed to resize disk on VM", "vmid", v.ID)
			}
		}

		err = db.UpdateVMStatus(v.ID, string(VMStatusStopped))
		if err != nil {
			logger.Error("Failed to update status of VM", "vmid", v.ID, "new_status", VMStatusStopped, "err", err)
		}

		logger.Debug("VM configured", "vm", vm)
	}
}

// updateVMs updates the status of VMs in the database based on their current status in Proxmox.
func updateVMs(cluster *gprox.Cluster) {
	logger.Debug("Updating VMs in worker")

	resources, err := getProxmoxResources(cluster, "vm")
	allVMStatus := []string{string(VMStatusRunning), string(VMStatusStopped), string(VMStatusSuspended)}

	activeVMs, err := db.GetAllActiveVMsWithUnknown()
	if err != nil {
		logger.Error("Can't get active VMs from DB", "err", err)
		return
	}

	// Map all vms to a map
	vmMap := make(map[uint64]*db.VM)
	for i := range activeVMs {
		vmMap[activeVMs[i].ID] = &activeVMs[i]
	}

	// Updates all DB VM's status
	for _, r := range resources {
		if r.Type != "qemu" {
			continue
		}

		// Check if the vm is managed by sasso, if not ignore
		vm, ok := vmMap[r.VMID]
		if !ok {
			continue
		}

		if vm.Status == string(VMStatusUnknown) {
			logger.Warn("VM changed status from unknown to a known status", "vmid", r.VMID, "new_status", r.Status)
			err := db.UpdateVMStatus(r.VMID, r.Status)
			if err != nil {
				logger.Error("Failed to update status of VM", "vmid", r.VMID, "new_status", r.Status, "err", err)
			}

			if vm.OwnerType == "Group" {
				err = notify.SendVMStatusUpdateNotificationToGroup(vm.OwnerID, vm.Name, r.Status)
			} else {
				err = notify.SendVMStatusUpdateNotification(vm.OwnerID, vm.Name, r.Status)
			}
			if err != nil {
				logger.Error("Failed to send VM status update notification", "vmid", r.VMID, "new_status", r.Status, "err", err)
			}

		} else if !slices.Contains(allVMStatus, r.Status) {
			vmStatusTimeMapEntry, exists := vmStatusTimeMap[r.VMID]

			timeToWait := 1 * time.Minute
			// VMs can be in the 'prelaunch' status during a backup, so we give it more time
			// before setting the status to unknown
			if exists && vmStatusTimeMapEntry.Value == "prelaunch" {
				timeToWait = 5 * time.Minute
			}

			if exists && time.Since(vmStatusTimeMapEntry.Time) > timeToWait && vmStatusTimeMapEntry.Value == r.Status {
				logger.Error("VM status unrecognised, setting status to unknown", "vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status)

				err := db.UpdateVMStatus(r.VMID, string(VMStatusUnknown))
				if err != nil {
					logger.Error("Failed to update status of VM", "vmid", r.VMID, "new_status", VMStatusUnknown, "err", err)
				}

				if vm.OwnerType == "Group" {
					err = notify.SendVMStatusUpdateNotificationToGroup(vm.OwnerID, vm.Name, string(VMStatusUnknown))
				} else {
					err = notify.SendVMStatusUpdateNotification(vm.OwnerID, vm.Name, string(VMStatusUnknown))
				}
				if err != nil {
					logger.Error("Failed to send VM status update notification", "vmid", r.VMID, "new_status", VMStatusUnknown, "err", err)
				}
			} else if !exists || vmStatusTimeMapEntry.Value != r.Status {
				t := time.Now()
				if exists {
					t = vmStatusTimeMapEntry.Time
				}
				vmStatusTimeMap[r.VMID] = stringTime{
					Value: r.Status,
					Time:  t,
				}
			}
		} else if r.Status != vm.Status && vm.UpdatedAt.Before(time.Now().Add(-1*time.Minute)) {
			logger.Warn("VM changed status on proxmox unexpectedly", "vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status)

			status := r.Status
			if !slices.Contains(allVMStatus, r.Status) {
				logger.Error("VM status not recognised, setting status to unknown", "vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status)
				status = string(VMStatusUnknown)
			}

			err := db.UpdateVMStatus(r.VMID, status)
			if err != nil {
				logger.Error("Failed to update status of VM", "vmid", r.VMID, "new_status", status, "err", err)
			}

			if vm.OwnerType == "Group" {
				err = notify.SendVMStatusUpdateNotificationToGroup(vm.OwnerID, vm.Name, status)
			} else {
				err = notify.SendVMStatusUpdateNotification(vm.OwnerID, vm.Name, status)
			}
			if err != nil {
				logger.Error("Failed to send VM status update notification", "vmid", r.VMID, "new_status", status, "err", err)
			}
		}
	}

	// Check if some VM that should be in proxmox is not present
	proxmoxVmsIDs := make([]uint64, len(resources))
	for i := range resources {
		proxmoxVmsIDs = append(proxmoxVmsIDs, resources[i].VMID)
	}

	slices.Sort(proxmoxVmsIDs)

	for i := range activeVMs {
		vmid := activeVMs[i].ID
		_, found := slices.BinarySearch(proxmoxVmsIDs, vmid)
		if found {
			continue
		}
		if activeVMs[i].Status == string(VMStatusUnknown) {
			continue
		}

		logger.Error("VM not found on proxmox but is on sasso. Setting status to unknown", "vmid", vmid, "status", activeVMs[i].Status)

		err := db.UpdateVMStatus(vmid, string(VMStatusUnknown))
		if err != nil {
			logger.Error("Failed to update status of VM", "vmid", vmid, "new_status", VMStatusUnknown, "err", err)
		}
	}
}

func createInterfaces(vmNodes map[uint64]string) {
	logger.Debug("Configuring interfaces in worker")

	interfaces, err := db.GetInterfacesWithStatus(string(InterfaceStatusPreCreating))
	if err != nil {
		logger.Error("Failed to get interfaces with 'pre-creating' status", "error", err)
		return
	}

	for _, iface := range interfaces {

		dbVM, err := db.GetVMByID(uint64(iface.VMID))
		if err != nil {
			logger.Error("Failed to get VM by ID for interface", "interface_id", iface.ID, "vmid", iface.VMID, "err", err)
			continue
		}

		if !slices.Contains(goodVMStatesForInterfacesManipulation, VMStatus(dbVM.Status)) {
			logger.Warn("Can't configure interface. VM not in a good state for interface manipulation", "vmid", iface.VMID, "interface_id", iface.ID, "vm_status", dbVM.Status)
			continue
		}

		nodeName, ok := vmNodes[uint64(iface.VMID)]
		if !ok {
			logger.Error("Can't configure interface. VM not found on cluster resources", "vmid", iface.VMID, "interface_id", iface.ID)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		node, err := client.Node(ctx, nodeName)
		cancel()
		if err != nil {
			logger.Error("Can't get node. Can't configure interface", "err", err, "vmid", iface.VMID, "interface_id", iface.ID)
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		vm, err := node.VirtualMachine(ctx, int(iface.VMID))
		cancel()
		if err != nil {
			logger.Error("Can't get VM. Can't configure VM", "err", err, "vmid", iface.VMID)
			continue
		}

		// TODO: Check if in the future the APIs will acctually support Nets maps
		// https://github.com/luthermonson/go-proxmox/issues/211
		// This is a temporary workaround.
		// At the moment we are using the samuelemusiani/go-proxmox fork
		mnets := vm.VirtualMachineConfig.Nets
		// mnets := map[net0:virtio=BC:24:11:D2:FA:F0,bridge=vmbr0,firewall=1 net1:virtio=BC:24:11:B6:1C:2A,bridge=sassoint,firewall=1]

		var snet = make([]int, len(mnets))
		var i int = 0
		for k := range mnets {
			tmp := strings.TrimPrefix(k, "net")
			tmpN, err := strconv.Atoi(tmp)
			if err != nil {
				continue
			}
			snet[i] = tmpN
			i++
		}
		slices.Sort(snet)
		logger.Debug("Current network interfaces on Proxmox VM", "mnets", mnets, "snet", snet)
		var firstEmptyIndex int = -1
		for i := range snet {
			if snet[i] != i {
				firstEmptyIndex = i
				break
			}
		}
		if firstEmptyIndex == -1 {
			firstEmptyIndex = len(snet)
		}

		vnet, err := db.GetNetByID(iface.VNetID)
		if err != nil {
			logger.Error("Failed to get net by ID", "interface_id", iface.ID, "net_id", iface.VNetID, "err", err)
			continue
		}

		v := fmt.Sprintf("virtio,bridge=%s,firewall=1", vnet.Name)
		if iface.VlanTag != 0 {
			v = fmt.Sprintf("%s,tag=%d", v, iface.VlanTag)
		}
		o := gprox.VirtualMachineOption{
			Name:  "net" + strconv.Itoa(firstEmptyIndex),
			Value: v,
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err := vm.Config(ctx, o)
		cancel()
		if err != nil {
			logger.Error("Failed to add network interface to Proxmox VM", "error", err)
			continue
		}
		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("Failed to wait for Proxmox task completion", "error", err)
			continue
		}

		if !isSuccessful {
			logger.Error("Failed to add network interface to Proxmox VM")
			continue
		}

		gatewayIpAddress := ipaddr.NewIPAddressString(iface.Gateway)
		gatewayIpAddressNoMask := gatewayIpAddress.GetAddress().WithoutPrefixLen().String()

		o2 := gprox.VirtualMachineOption{
			Name:  "ipconfig" + strconv.Itoa(firstEmptyIndex),
			Value: fmt.Sprintf("ip=%s,gw=%s", iface.IPAdd, gatewayIpAddressNoMask),
		}

		logger.Debug("Configuring network interface on Proxmox VM", "option", o2)

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err = vm.Config(ctx, o2)
		cancel()
		if err != nil {
			logger.Error("Failed to configure network interface on Proxmox VM", "error", err)
			continue
		}
		isSuccessful, err = waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("Failed to wait for Proxmox task completion", "error", err)
			continue
		}
		if !isSuccessful {
			logger.Error("Failed to configure network interface on Proxmox VM")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		err = vm.RegenerateCloudInitImage(ctx)
		cancel()
		if err != nil {
			logger.Error("Failed to regenerate cloud-init image on Proxmox VM", "error", err)
		}

		iface.LocalID = uint(firstEmptyIndex)
		iface.Status = string(InterfaceStatusReady)
		err = db.UpdateInterface(&iface)
		if err != nil {
			logger.Error("Failed to update interface status to ready", "interface", iface, "err", err)
			continue
		}
	}
}

func deleteInterfaces(vmNodes map[uint64]string) {
	logger.Debug("Configuring interfaces in worker")

	interfaces, err := db.GetInterfacesWithStatus(string(InterfaceStatusPreDeleting))
	if err != nil {
		logger.Error("Failed to get interfaces with 'pre-creating' status", "error", err)
		return
	}

	for _, iface := range interfaces {
		nodeName, ok := vmNodes[uint64(iface.VMID)]
		if !ok {
			logger.Error("Can't configure interface. VM not found on cluster resources", "vmid", iface.VMID, "interface_id", iface.ID)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		node, err := client.Node(ctx, nodeName)
		cancel()
		if err != nil {
			logger.Error("Can't get node. Can't configure interface", "err", err, "vmid", iface.VMID, "interface_id", iface.ID)
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		vm, err := node.VirtualMachine(ctx, int(iface.VMID))
		cancel()
		if err != nil {
			logger.Error("Can't get VM. Can't configure VM", "err", err, "vmid", iface.VMID)
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err := vm.Config(ctx, gprox.VirtualMachineOption{
			Name:  "delete",
			Value: fmt.Sprintf("net%d", iface.LocalID),
		})
		cancel()
		if err != nil {
			logger.Error("Failed to remove network interface from Proxmox VM", "error", err)
			continue
		}

		err = db.UpdateInterfaceStatus(iface.ID, string(InterfaceStatusDeleting))
		if err != nil {
			logger.Error("Failed to update interface status to deleting", "interface", iface, "err", err)
			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("Failed to wait for Proxmox task completion", "error", err)
			continue
		}
		if !isSuccessful {
			logger.Error("Failed to remove network interface from Proxmox VM")
			continue
		}

		err = db.DeleteInterfaceByID(iface.ID)
		if err != nil {
			logger.Error("Failed to delete interface from DB", "interface", iface, "err", err)
			continue
		}
	}
}

func deleteBackups(mapVMContent map[uint64]string) {
	logger.Debug("Deleting backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeDelete)
	if err != nil {
		logger.Error("Failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeDelete, "error", err)
		return
	}

	for _, r := range bkr {
		slog.Debug("Deleting backup", "id", r.ID)

		if r.Volid == nil {
			slog.Error("Can't delete backup. Volid is nil", "id", r.ID)
			continue
		}

		nodeName, ok := mapVMContent[uint64(r.VMID)]
		if !ok {
			logger.Error("Can't delete backup. Not found on cluster resources", "vmid", r.VMID)
			continue
		}

		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			logger.Error("Failed to get Proxmox node", "node", nodeName, "error", err)
			continue
		}

		storage, err := getProxmoxStorage(node, cBackup.Storage)
		if err != nil {
			logger.Error("Failed to get Proxmox storage", "storage", cBackup.Storage, "error", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t, err := storage.DeleteContent(ctx, *r.Volid)
		defer cancel()
		if err != nil {
			logger.Error("failed to delete content", "error", err)
			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("Failed to wait for Proxmox task completion", "error", err)
			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.Error("Failed to update backup request status to failed", "id", r.ID, "error", err)
			}
			continue
		}

		var status string
		if isSuccessful {
			status = BackupRequestStatusCompleted
		} else {
			status = BackupRequestStatusFailed
		}
		err = db.UpdateBackupRequestStatus(r.ID, status)
		if err != nil {
			logger.Error("Failed to update backup request status", "status", status, "id", r.ID, "error", err)
		}
	}
}

func restoreBackups(mapVMContent map[uint64]string) {
	logger.Debug("Restoring backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeRestore)
	if err != nil {
		logger.Error("Failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeRestore, "error", err)
		return
	}

	for _, r := range bkr {
		slog.Debug("Restoring backup", "id", r.ID)

		if r.Volid == nil {
			slog.Error("Can't restore backup. Volid is nil", "id", r.ID)
			continue
		}

		nodeName, ok := mapVMContent[uint64(r.VMID)]
		if !ok {
			logger.Error("Can't delete backup. Not found on cluster resources", "vmid", r.VMID)
			continue
		}

		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			logger.Error("Failed to get Proxmox node", "node", nodeName, "error", err)
			continue
		}

		o1 := gprox.VirtualMachineOption{
			Name:  "force",
			Value: "1",
		}
		o2 := gprox.VirtualMachineOption{
			Name:  "archive",
			Value: r.Volid,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t, err := node.NewVirtualMachine(ctx, int(r.VMID), o1, o2)
		defer cancel()
		if err != nil {
			logger.Error("failed to create new vm", "error", err)
			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("Failed to wait for Proxmox task completion", "error", err)
			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.Error("Failed to update backup request status to failed", "id", r.ID, "error", err)
			}
			continue
		}

		var status string
		if isSuccessful {
			status = BackupRequestStatusCompleted
		} else {
			status = BackupRequestStatusFailed
		}
		err = db.UpdateBackupRequestStatus(r.ID, status)
		if err != nil {
			logger.Error("Failed to update backup request status", "status", status, "id", r.ID, "error", err)
		}
	}
}

func createBackups(mapVMContent map[uint64]string) {
	logger.Debug("Creating backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeCreate)
	if err != nil {
		logger.Error("Failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeCreate, "error", err)
		return
	}

	for _, r := range bkr {
		slog.Debug("Creating backup", "id", r.ID)

		nodeName, ok := mapVMContent[uint64(r.VMID)]
		if !ok {
			logger.Error("Can't delete backup. Not found on cluster resources", "vmid", r.VMID)
			continue
		}

		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			logger.Error("Failed to get Proxmox node", "node", nodeName, "error", err)
			continue
		}

		notes, err := generateBackNotes(r.Name, r.Notes, r.OwnerID, r.OwnerType)
		if err != nil {
			logger.Error("Failed to generate backup notes", "error", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t, err := node.Vzdump(ctx, &gprox.VirtualMachineBackupOptions{
			Storage:       cBackup.Storage,
			VMID:          uint64(r.VMID),
			Mode:          "snapshot",
			Remove:        false,
			Compress:      "zstd",
			NotesTemplate: notes,
		})
		cancel()
		if err != nil {
			slog.Error("failed to create vzdump", "error", err)
			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("Failed to wait for Proxmox task completion", "error", err)
			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.Error("Failed to update backup request status to failed", "id", r.ID, "error", err)
			}
			continue
		}

		var status string
		if isSuccessful {
			status = BackupRequestStatusCompleted
		} else {
			status = BackupRequestStatusFailed
		}
		err = db.UpdateBackupRequestStatus(r.ID, status)
		if err != nil {
			logger.Error("Failed to update backup request status", "status", status, "id", r.ID, "error", err)
		}
	}
}

func configureSSHKeys(vmNodes map[uint64]string) {
	logger.Debug("Configuring SSH keys in worker")

	states := []string{string(VMStatusStopped), string(VMStatusRunning), string(VMStatusSuspended)}

	ssht := db.GetLastSSHKeyUpdate()
	vmt, err := db.GetTimeOfLastCreatedVMWithStates(states)
	if err != nil {
		logger.Error("Failed to get time of last created VM with states", "error", err)
		return
	}

	// Every 6 hours we force a reconfiguration of SSH keys
	if !lastConfigureSSHKeysTime.Before(time.Now().Add(-6*time.Hour)) &&
		lastConfigureSSHKeysTime.After(ssht) && lastConfigureSSHKeysTime.After(vmt) {
		logger.Debug("No need to configure SSH keys. No new SSH keys or VMs")
		return
	}

	// TODO: We could optimize this further by checking why the ssh keys table
	// changed and only updating the VMs of the users that had changes (unless
	// global keys changed)

	vms, err := db.GetVMsWithStates(states)
	if err != nil {
		logger.Error("Failed to get VMs with 'stopped' status", "error", err)
		return
	}

	for _, v := range vms {
		nodeName, ok := vmNodes[v.ID]
		if !ok {
			logger.Error("Can't configure VM. Not found on cluster resources", "vmid", v.ID)
			continue
		}
		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			continue
		}

		vm, err := getProxmoxVM(node, int(v.ID))
		if err != nil {
			continue
		}

		var sshKeys []db.SSHKey
		if v.OwnerType == "Group" {
			sshKeys, err = db.GetSSHKeysByGroupID(v.OwnerID)
		} else {
			sshKeys, err = db.GetSSHKeysByUserID(v.OwnerID)
		}
		if err != nil {
			logger.Error("Failed to get SSH keys for user", "vmid", v.ID, "ownerID", v.OwnerID, "ownerType", v.OwnerType, "err", err)
			continue
		}

		if v.IncludeGlobalSSHKeys {
			globalKeys, err := db.GetGlobalSSHKeys()
			if err != nil {
				logger.Error("Failed to get global SSH keys", "vmid", v.ID, "err ", err)
				continue
			}

			sshKeys = append(sshKeys, globalKeys...)
		}

		var keys strings.Builder
		for i := range sshKeys {
			keys.WriteString(sshKeys[i].Key)
			keys.WriteString("\n")
		}
		cloudInitKeys := strings.ReplaceAll(url.QueryEscape(keys.String()), "+", "%20")

		if vm.VirtualMachineConfig.SSHKeys == cloudInitKeys {
			continue
		}

		sshOption := gprox.VirtualMachineOption{
			Name:  "sshkeys",
			Value: cloudInitKeys,
		}
		isSuccessful, err := configureVM(vm, sshOption)
		if err != nil {
			logger.Error("Failed to set ssh keys on VM", "vmid", v.ID, "err", err)
			return
		}
		logger.Debug("Task finished", "isSuccessful", isSuccessful)
		if !isSuccessful {
			logger.Error("Failed to set ssh keys on VM", "vmid", v.ID)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = vm.RegenerateCloudInitImage(ctx)
		cancel()
		if err != nil {
			logger.Error("Failed to regenerate cloud init image on VM", "vmid", v.ID, "err", err)
		}

		if v.OwnerType == "Group" {
			err = notify.SendSSHKeysChangedOnVMToGroup(v.OwnerID, v.Name)
			if err != nil {
				logger.Error("Failed to send SSH keys changed notification to group", "vmid", v.ID, "err", err)
			}
		}
	}

	lastConfigureSSHKeysTime = time.Now()
}

func enforceVMLifetimes() {
	t := time.Now().AddDate(0, 3, 0) // 3 months from now
	vms, err := db.GetVMsWithLifetimesLessThan(t)
	if err != nil {
		logger.Error("Failed to get VMs with lifetimes less than", "time", t, "error", err)
		return
	}

	fn := func(n int64) func(not db.VMExpirationNotification) bool {
		return func(not db.VMExpirationNotification) bool {
			return int64(not.DaysBefore) == n
		}
	}

	for _, v := range vms {
		notifications, err := db.GetVMExpirationNotificationsByVMID(v.ID)
		if err != nil {
			logger.Error("Failed to get VM expiration notifications for VM", "vmid", v.ID, "error", err)
			continue
		}

		if v.LifeTime.Before(time.Now().Add(-7 * 24 * time.Hour)) {
			// The VM expired more than 7 days ago, we delete it
			err := deleteVMBypass(v.ID)
			if err != nil {
				logger.Error("Failed to delete expired VM", "vmid", v.ID, "error", err)
				continue
			}

			if v.OwnerType == "Group" {
				err = notify.SendVMEliminatedNotificationToGroup(v.OwnerID, v.Name)
			} else {
				err = notify.SendVMEliminatedNotification(v.OwnerID, v.Name)
			}
			if err != nil {
				logger.Error("Failed to send VM eliminated notification", "vmid", v.ID, "error", err)
			}
		} else if v.LifeTime.Before(time.Now()) && v.Status != string(VMStatusStopped) {
			// The VM expired, but less than 7 days ago, we send the last notification
			// and stop the VM if it is running
			err := changeVMStatusBypass(v.ID, "stop")
			if err != nil {
				logger.Error("Failed to stop expired VM", "vmid", v.ID, "error", err)
				continue
			}

			if v.OwnerType == "Group" {
				err = notify.SendVMStoppedNotificationToGroup(v.OwnerID, v.Name)
			} else {
				err = notify.SendVMStoppedNotification(v.OwnerID, v.Name)
			}
			if err != nil {
				logger.Error("Failed to send VM stopped notification", "vmid", v.ID, "error", err)
			}
		} else {
			for _, i := range []int64{1, 2, 4, 7, 15, 30, 60, 90} {
				if slices.ContainsFunc(notifications, fn(i)) {
					break
				}
				if v.LifeTime.Before(time.Now().AddDate(0, 0, int(i))) && v.LifeTime.After(v.CreatedAt.AddDate(0, 0, int(i))) {
					// Send notification for i day before expiration
					if v.OwnerType == "Group" {
						err = notify.SendVMExpirationNotificationToGroup(v.OwnerID, v.Name, int(i))
					} else {
						err = notify.SendVMExpirationNotification(v.OwnerID, v.Name, int(i))
					}
					if err != nil {
						logger.Error("Failed to send VM expiration notification", "vmid", v.ID, "days_before", i, "error", err)
						continue
					}
					_, err = db.NewVMExpirationNotification(v.ID, uint(i))
					if err != nil {
						logger.Error("Failed to create VM expiration notification", "vmid", v.ID, "days_before", i, "error", err)
					}
					break
				}
			}
		}
	}
}
