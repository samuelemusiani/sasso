// Sasso will write to the DB what the current state should look like. The
// worker will read the DB and take care of all the operations that needs
// to be done

package proxmox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	gprox "github.com/luthermonson/go-proxmox"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/notify"
)

type stringTime struct {
	Value string
	Time  time.Time
}

var (
	vmStatusTimeMap        map[uint64]stringTime = make(map[uint64]stringTime)
	vmLastTimePrelaunchMap map[uint64]time.Time  = make(map[uint64]time.Time)

	// Last time a VM failed to change status. This is used in the
	// enforceVMLifetimes function
	vmFailedChangeStatus map[uint64]time.Time = make(map[uint64]time.Time)

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

	if err != nil && !errors.Is(err, context.Canceled) {
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
			logger.Error("failed to get Proxmox cluster", "error", err)
			time.Sleep(5 * time.Second)

			continue
		}

		workerCycleDurationObserve("create_vnets", func() { createVNets(cluster) })
		workerCycleDurationObserve("delete_vnets", func() { deleteVNets(cluster) })
		workerCycleDurationObserve("configure_vnets", func() { configureVNets(cluster) })
		workerCycleDurationObserve("update_vnets", func() { updateVNets(cluster) })

		workerCycleDurationObserve("create_vms", func() { createVMs() })
		workerCycleDurationObserve("update_vms", func() { updateVMs(cluster) })

		workerCycleDurationObserve("lifetime_vms", func() { enforceVMLifetimes() })

		vmNodes, err := mapVMIDToProxmoxNodes(cluster)
		if err != nil {
			logger.Error("failed to map VMID to Proxmox nodes", "error", err)
			time.Sleep(5 * time.Second)

			continue
		}

		workerCycleDurationObserve("delete_vms", func() { deleteVMs(vmNodes) })
		workerCycleDurationObserve("configure_ssh_keys", func() { configureSSHKeys(vmNodes) })
		workerCycleDurationObserve("configure_vms", func() { configureVMs(vmNodes) })

		workerCycleDurationObserve("create_interfaces", func() { createInterfaces(vmNodes) })
		workerCycleDurationObserve("delete_interfaces", func() { deleteInterfaces(vmNodes) })
		workerCycleDurationObserve("configure_interfaces", func() { configureInterfaces(vmNodes) })

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
		logger.Error("failed to count VMs in DB", "error", err)
	} else {
		objectCountSet("vms", vmsCount)
	}

	interfacesCount, err := db.CountInterfaces()
	if err != nil {
		logger.Error("failed to count interfaces in DB", "error", err)
	} else {
		objectCountSet("interfaces", interfacesCount)
	}

	netsCount, err := db.CountVNets()
	if err != nil {
		logger.Error("failed to count VNets in DB", "error", err)
	} else {
		objectCountSet("vnets", netsCount)
	}

	countPortFowards, err := db.CountPortForwards()
	if err != nil {
		logger.Error("failed to count port forwards in DB", "error", err)
	} else {
		objectCountSet("port_forwards", countPortFowards)
	}
}

func createVNets(cluster *gprox.Cluster) {
	logger.Debug("Creating VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VMStatusPreCreating))
	if err != nil {
		logger.Error("failed to get VNets with 'pre-creating' status", "error", err)

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
			logger.Error("failed to create VNet in Proxmox", "vnet", v.Name, "error", err)

			continue
		}

		err = db.UpdateVNetStatus(v.ID, string(VNetStatusCreating))
		if err != nil {
			logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusCreating, "err", err)

			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)

	cancel()

	if err != nil {
		logger.Error("failed to apply SDN changes in Proxmox", "error", err)

		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if err != nil {
		logger.Error("failed to wait for Proxmox task completion", "error", err)

		return
	}

	if !isSuccessful {
		logger.Error("failed to apply SDN changes in Proxmox")
		// Set all VNets status to 'unknown'
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}
		}

		return
	} else {
		logger.Debug("SDN changes applied successfully")

		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusReady))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusReady, "err", err)
			}
		}
	}
}

func deleteVNets(cluster *gprox.Cluster) {
	logger.Debug("Deleting VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VNetStatusPreDeleting))
	if err != nil {
		logger.Error("failed to get VNets with 'pre-deleting' status", "error", err)

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
			logger.Error("failed to delete VNet from Proxmox", "vnet", v.Name, "error", err)

			continue
		}

		err = db.UpdateVNetStatus(v.ID, string(VNetStatusDeleting))
		if err != nil {
			logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusDeleting, "err", err)

			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)

	cancel()

	if err != nil {
		logger.Error("failed to apply SDN changes in Proxmox", "error", err)

		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if err != nil {
		logger.Error("failed to wait for Proxmox task completion", "error", err)

		return
	}

	if isSuccessful {
		logger.Debug("SDN changes applied successfully")

		for _, v := range vnets {
			err = db.DeleteNetByID(v.ID)
			if err != nil {
				logger.Error("failed to delete VNet from DB", "vnet", v.Name, "err", err)
			}
		}
	} else {
		logger.Error("failed to apply SDN changes in Proxmox")

		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}
		}
	}
}

// createVMs creates VMs from proxmox that are in the 'pre-creating' status.
func createVMs() {
	logger.Debug("Creating VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreCreating))
	if err != nil {
		logger.Error("failed to get VMs with 'creating' status", "error", err)

		return
	}

	if len(vms) == 0 {
		return
	}

	node, err := getProxmoxNode(client, cTemplate.Node)
	if err != nil {
		return
	}

	templateVM, err := getProxmoxVM(node, cTemplate.VMID)
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
				logger.Error("failed to get unique owner ID for VM naming", "vmid", v.ID, "err", err)

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
		_, task, err := templateVM.Clone(ctx, &cloningOptions)

		cancel()

		if err != nil {
			logger.Error("failed to clone VM", "vmid", v.ID, "error", err)

			continue
		}

		err = db.UpdateVMStatus(v.ID, string(VMStatusCreating))
		if err != nil {
			logger.Error("failed to update status of VM", "vmid", v.ID, "new_status", VMStatusCreating, "err", err)
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(task)
		if err != nil {
			logger.Error("failed to wait for Proxmox task completion", "err", err, "vmid", v.ID)

			continue
		}

		if isSuccessful {
			err = db.UpdateVMStatus(v.ID, string(VMStatusPreConfiguring))
			if err != nil {
				logger.Error("failed to update status of VM", "vmid", v.ID, "new_status", VMStatusStopped, "err", err)
			}
		} else {
			// We could set the status as pre-creating to trigger a recreation, but
			// for now we just set it to unknown
			err = db.UpdateVMStatus(v.ID, string(VMStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VM", "vmid", v.ID, "new_status", VMStatusUnknown, "err", err)
			}
		}
	}
}

// deleteVMs deletes VMs from proxmox that are in the 'pre-deleting' status.
func deleteVMs(vmsLocation map[uint64]string) {
	logger.Debug("Deleting VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreDeleting))
	if err != nil {
		logger.Error("failed to get VMs with 'deleting' status", "error", err)

		return
	}

	for _, v := range vms {
		logger.Debug("Deleting VM", "vmid", v.ID)

		err := db.DeleteAllInterfacesByVMID(v.ID)
		if err != nil {
			logger.Error("failed to delete interfaces for VM", "vmid", v.ID, "err", err)
		}

		nodeName, ok := vmsLocation[v.ID]
		if !ok {
			logger.Error("Can't delete VM. Not found on cluster resources", "vmid", v.ID)

			// If the VM is not found on Proxmox, we just delete it from the DB
			err = db.DeleteVMByID(v.ID)
			if err != nil {
				logger.Error("failed to delete VM", "vmid", v.ID, "err", err)
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
			logger.Warn("failed to update status of VM", "vmid", v.ID, "new_status", VMStatusDeleting, "err", err)
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(task)
		if err != nil {
			logger.Error("failed to wait for Proxmox task completion", "err", err, "vmid", v.ID)

			continue
		}

		cancel()

		if isSuccessful {
			err = db.DeleteVMByID(v.ID)
			if err != nil {
				logger.Error("failed to delete VM", "vmid", v.ID, "err", err)
			}
		} else {
			// We could set the status as pre-creating to trigger a recreation, but
			// for now we just set it to unknown
			err = db.UpdateVMStatus(v.ID, string(VMStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VM", "vmid", v.ID, "new_status", VMStatusUnknown, "err", err)
			}
		}
	}
}

func configureVNets(cluster *gprox.Cluster) {
	logger.Debug("Configuring VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VNetStatusReconfiguring))
	if err != nil {
		logger.Error("failed to get VNets with status", "status", VNetStatusReconfiguring, "error", err)

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
			logger.Error("failed to get VNet from Proxmox", "vnet", v.Name, "error", err)

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
			logger.Error("failed to update VNet in Proxmox", "vnet", v.Name, "error", err)

			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)

	cancel()

	if err != nil {
		logger.Error("failed to apply SDN changes in Proxmox", "error", err)

		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if err != nil {
		logger.Error("failed to wait for Proxmox task completion", "error", err)

		return
	}

	if isSuccessful {
		logger.Debug("SDN changes applied successfully")

		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusReady))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusReady, "err", err)
			}
		}
	} else {
		logger.Error("failed to apply SDN changes in Proxmox")

		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}
		}
	}
}

func updateVNets(cluster *gprox.Cluster) {
	logger.Debug("Updating VNets in worker")

	dbVNets, err := db.GetVNetsWithStatus(string(VNetStatusReady))
	if err != nil {
		logger.Error("failed to get VNets with 'pre-creating' status", "error", err)

		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pVNets, err := cluster.SDNVNets(ctx)

	cancel()

	if err != nil {
		logger.Error("failed to get VNets from Proxmox", "error", err)

		return
	}

	nameToPVNet := make(map[string]*gprox.VNet)
	for _, pn := range pVNets {
		nameToPVNet[pn.Name] = pn
	}

	tagToPVNet := make(map[uint32]*gprox.VNet)
	for _, pn := range pVNets {
		tagToPVNet[pn.Tag] = pn
	}

	for _, v := range dbVNets {
		pvn, ok := nameToPVNet[v.Name]
		if !ok {
			logger.Warn("VNet not found in Proxmox. Setting status to unknown", "vnet", v.Name)

			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}

			continue
		}

		if pvn.Tag != v.Tag {
			logger.Warn("VNet tag mismatch. Setting status to unknown", "vnet", v.Name, "db_tag", v.Tag, "proxmox_tag", pvn.Tag)

			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}

			continue
		}

		intToBool := func(i int) bool {
			return i == 1
		}

		if v.VlanAware != intToBool(pvn.VlanAware) {
			logger.Warn("VNet vlan_aware mismatch. Setting status to unknown", "vnet", v.Name, "db_vlan_aware", v.VlanAware, "proxmox_vlan_aware", intToBool(pvn.VlanAware))

			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.Error("failed to update status of VNet", "vnet", v.Name, "new_status", VNetStatusUnknown, "err", err)
			}

			continue
		}
	}
}

// This function configures VMs that are in the 'pre-configuring' status.
// Configuration includes setting the number of cores, RAM and disk size
func configureVMs(vmNodes map[uint64]string) {
	logger.Debug("Configuring VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreConfiguring))
	if err != nil {
		logger.Error("failed to get VMs with 'pre-configuring' status", "error", err)

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
				logger.Error("failed to set cores on VM", "vmid", v.ID, "err", err)

				return
			}

			logger.Debug("Task finished", "isSuccessful", isSuccessful)

			if !isSuccessful {
				logger.Error("failed to set cores on VM", "vmid", v.ID)
			}
		}

		if uint(vm.VirtualMachineConfig.Memory) != v.RAM {
			ramOption := gprox.VirtualMachineOption{
				Name:  "memory",
				Value: v.RAM,
			}

			isSuccessful, err := configureVM(vm, ramOption)
			if err != nil {
				logger.Error("failed to set ram on VM", "vmid", v.ID, "err", err)

				continue
			}

			logger.Debug("Task finished", "isSuccessful", isSuccessful)

			if !isSuccessful {
				logger.Error("failed to set ram on VM", "vmid", v.ID)
			}
		}

		scsi0, ok := vm.VirtualMachineConfig.SCSIs["scsi0"]
		if !ok {
			logger.Error("failed to find SCSI0 on VM", "vmid", v.ID)

			continue
		}

		st, err := parseStorageFromString(scsi0)
		if err != nil {
			logger.Error("failed to parse storage on SCSI0", "vmid", v.ID, "scsi0", scsi0)

			continue
		}

		if st.Size < v.Disk {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			diff := v.Disk - st.Size
			t, err := vm.ResizeDisk(ctx, "scsi0", fmt.Sprintf("+%dG", diff))

			cancel()

			if err != nil {
				logger.Error("failed to set resize disk on VM", "vmid", v.ID, "err", err)

				continue
			}

			isSuccessful, err := waitForProxmoxTaskCompletion(t)
			if err != nil {
				logger.Error("failed to wait for resize disk task completion", "vmid", v.ID, "err", err)

				continue
			}

			logger.Debug("Task finished", "isSuccessful", isSuccessful)

			if !isSuccessful {
				logger.Error("failed to resize disk on VM", "vmid", v.ID)
			}
		}

		// If a VM needs to be reconfigured (for example changing cores, RAM or disk),
		// it is put in the 'pre-configuring' status. After the configuration is done,
		// the vm must be set to the old status, but we don't save it anywhere.
		// For new created VMs, we just set the status to 'stopped'.
		// For other VMs, we try to set it to the acctual status in Proxmox, but
		// if the status is not recognised, we set it to 'stopped'.
		// (This is not a huge issue, because the updateVMs function will eventually
		// correct the status)
		vmStates := []string{string(VMStatusRunning), string(VMStatusStopped), string(VMStatusPaused)}

		var newStatus string
		if slices.Contains(vmStates, vm.Status) {
			newStatus = vm.Status
		} else {
			newStatus = string(VMStatusStopped)
		}

		err = db.UpdateVMStatus(v.ID, newStatus)
		if err != nil {
			logger.Error("failed to update status of VM", "vmid", v.ID, "new_status", VMStatusStopped, "err", err)
		}

		logger.Debug("VM configured", "vm", vm)
	}
}

// updateVMs updates the status of VMs in the database based on their current status in Proxmox.
func updateVMs(cluster *gprox.Cluster) {
	logger.Debug("Updating VMs in worker")

	resources, err := getProxmoxResources(cluster, "vm")
	if err != nil {
		logger.Error("failed to get Proxmox resources", "error", err)

		return
	}

	allVMStatus := []string{string(VMStatusRunning), string(VMStatusStopped), string(VMStatusPaused)}

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

		statusInSlices := slices.Contains(allVMStatus, r.Status)
		// If the VMs status becomes normal we need to delete it from the map
		delete(vmStatusTimeMap, r.VMID)

		lastTimeVMWasPrelaunch, hasLastTimeVMWasPrelaunch := vmLastTimePrelaunchMap[r.VMID]
		if !hasLastTimeVMWasPrelaunch {
			lastTimeVMWasPrelaunch = time.Time{}
		}

		switch {
		case vm.Status == string(VMStatusUnknown) && statusInSlices:
			logger.Warn("VM changed status from unknown to a known status", "vmid", r.VMID, "new_status", r.Status)

			err := db.UpdateVMStatus(r.VMID, r.Status)
			if err != nil {
				logger.Error("failed to update status of VM", "vmid", r.VMID, "new_status", r.Status, "err", err)
			}

			if vm.OwnerType == "Group" {
				err = notify.SendVMStatusUpdateNotificationToGroup(vm.OwnerID, vm.Name, r.Status)
			} else {
				err = notify.SendVMStatusUpdateNotification(vm.OwnerID, vm.Name, r.Status)
			}

			if err != nil {
				logger.Error("failed to send VM status update notification", "vmid", r.VMID, "new_status", r.Status, "err", err)
			}

			delete(vmStatusTimeMap, r.VMID)
		case !statusInSlices:
			vmStatusTimeMapEntry, exists := vmStatusTimeMap[r.VMID]

			timeToWait := 1 * time.Minute
			// VMs can be in the 'prelaunch' status during a backup, so we give it more time
			// before setting the status to unknown
			if exists && vmStatusTimeMapEntry.Value == "prelaunch" {
				timeToWait = 5 * time.Minute
			}

			if r.Status == "prelaunch" {
				vmLastTimePrelaunchMap[r.VMID] = time.Now()
			}

			if exists && time.Since(vmStatusTimeMapEntry.Time) > timeToWait && vmStatusTimeMapEntry.Value == r.Status {
				logger.Error("VM status unrecognised, setting status to unknown", "vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status)

				err := db.UpdateVMStatus(r.VMID, string(VMStatusUnknown))
				if err != nil {
					logger.Error("failed to update status of VM", "vmid", r.VMID, "new_status", VMStatusUnknown, "err", err)
				}

				if vm.OwnerType == "Group" {
					err = notify.SendVMStatusUpdateNotificationToGroup(vm.OwnerID, vm.Name, string(VMStatusUnknown))
				} else {
					err = notify.SendVMStatusUpdateNotification(vm.OwnerID, vm.Name, string(VMStatusUnknown))
				}

				if err != nil {
					logger.Error("failed to send VM status update notification", "vmid", r.VMID, "new_status", VMStatusUnknown, "err", err)
				}

				delete(vmStatusTimeMap, r.VMID)
			} else if !exists || vmStatusTimeMapEntry.Value != r.Status {
				t := time.Now()
				// if exists {
				// 	t = vmStatusTimeMapEntry.Time
				// }
				vmStatusTimeMap[r.VMID] = stringTime{
					Value: r.Status,
					Time:  t,
				}
			}
		case r.Status != vm.Status &&
			vm.UpdatedAt.Before(time.Now().Add(-1*time.Minute)) && // Avoid status flapping right after a status change from the APIs
			lastTimeVMWasPrelaunch.Before(time.Now().Add(-5*time.Minute)): // Avoid status flapping right after a prelaunch
			logger.Warn("VM changed status on proxmox unexpectedly", "vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status)

			status := r.Status
			if !statusInSlices {
				logger.Error("VM status not recognised, setting status to unknown", "vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status)

				status = string(VMStatusUnknown)
			}

			err := db.UpdateVMStatus(r.VMID, status)
			if err != nil {
				logger.Error("failed to update status of VM", "vmid", r.VMID, "new_status", status, "err", err)
			}

			if vm.OwnerType == "Group" {
				err = notify.SendVMStatusUpdateNotificationToGroup(vm.OwnerID, vm.Name, status)
			} else {
				err = notify.SendVMStatusUpdateNotification(vm.OwnerID, vm.Name, status)
			}

			if err != nil {
				logger.Error("failed to send VM status update notification", "vmid", r.VMID, "new_status", status, "err", err)
			}

			delete(vmStatusTimeMap, r.VMID)
		}
	}

	// Check if some VM that should be in proxmox is not present
	proxmoxVmsIDs := make([]uint64, 0, len(resources))
	for _, r := range resources {
		proxmoxVmsIDs = append(proxmoxVmsIDs, r.VMID)
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
			logger.Error("failed to update status of VM", "vmid", vmid, "new_status", VMStatusUnknown, "err", err)
		}
	}
}

func createInterfaces(vmNodes map[uint64]string) {
	logger.Debug("Creating interfaces in worker")

	interfaces, err := db.GetInterfacesWithStatus(string(InterfaceStatusPreCreating))
	if err != nil {
		logger.Error("failed to get interfaces with 'pre-creating' status", "error", err)

		return
	}

	for _, iface := range interfaces {
		dbVM, err := db.GetVMByID(uint64(iface.VMID))
		if err != nil {
			logger.Error("failed to get VM by ID for interface", "interface_id", iface.ID, "vmid", iface.VMID, "err", err)

			continue
		}

		if !slices.Contains(goodVMStatesForInterfacesManipulation, VMStatus(dbVM.Status)) {
			logger.Warn("Can't create configure interface. VM not in a good state for interface manipulation", "vmid", iface.VMID, "interface_id", iface.ID, "vm_status", dbVM.Status)

			continue
		}

		nodeName, ok := vmNodes[uint64(iface.VMID)]
		if !ok {
			logger.Error("Can't configure interface. VM not found on cluster resources", "vmid", iface.VMID, "interface_id", iface.ID)

			continue
		}

		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			logger.Error("Can't get node. Can't configure interface", "err", err, "vmid", iface.VMID, "interface_id", iface.ID)

			continue
		}

		vm, err := getProxmoxVM(node, int(iface.VMID))
		if err != nil {
			logger.Error("Can't get VM. Can't configure VM", "err", err, "vmid", iface.VMID)

			continue
		}

		mnets := vm.VirtualMachineConfig.Nets
		// mnets := map[net0:virtio=BC:24:11:D2:FA:F0,bridge=vmbr0,firewall=1 net1:virtio=BC:24:11:B6:1C:2A,bridge=sassoint,firewall=1]

		// To avoid increasing the netX index indefinitely, we find the first
		// empty index
		snet := make([]int, len(mnets))
		i := 0

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

		firstEmptyIndex := -1

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
			logger.Error("failed to get net by ID", "interface_id", iface.ID, "net_id", iface.VNetID, "err", err)

			continue
		}

		v := "virtio,bridge=" + vnet.Name

		if cClone.EnableFirewall {
			v += ",firewall=1"
		}

		if cClone.MTU.Set {
			value := cClone.MTU.MTU
			if cClone.MTU.SameAsBridge {
				value = 1
			}

			v = fmt.Sprintf("%s,mtu=%d", v, value)
		}

		if iface.VlanTag != 0 {
			v = fmt.Sprintf("%s,tag=%d", v, iface.VlanTag)
		}

		o := gprox.VirtualMachineOption{
			Name:  "net" + strconv.Itoa(firstEmptyIndex),
			Value: v,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t, err := vm.Config(ctx, o)

		cancel()

		if err != nil {
			logger.Error("failed to add network interface to Proxmox VM", "error", err)

			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			continue
		}

		if !isSuccessful {
			logger.Error("failed to add network interface to Proxmox VM")

			continue
		}

		gatewayIPAddress := ipaddr.NewIPAddressString(iface.Gateway)
		gatewayIPAddressNoMask := gatewayIPAddress.GetAddress().WithoutPrefixLen().String()

		o2 := gprox.VirtualMachineOption{
			Name:  "ipconfig" + strconv.Itoa(firstEmptyIndex),
			Value: "ip=" + iface.IPAdd,
		}

		if iface.Gateway != "" {
			o2.Value = fmt.Sprintf("%s,gw=%s", o2.Value, gatewayIPAddressNoMask)
		}

		logger.Debug("Configuring network interface on Proxmox VM", "option", o2)

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err = vm.Config(ctx, o2)

		cancel()

		if err != nil {
			logger.Error("failed to configure network interface on Proxmox VM", "error", err)

			continue
		}

		isSuccessful, err = waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			continue
		}

		if !isSuccessful {
			logger.Error("failed to configure network interface on Proxmox VM")

			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		err = vm.RegenerateCloudInitImage(ctx)

		cancel()

		if err != nil {
			logger.Error("failed to regenerate cloud-init image on Proxmox VM", "error", err)
		}

		iface.LocalID = uint(firstEmptyIndex)
		iface.Status = string(InterfaceStatusReady)

		err = db.UpdateInterface(&iface)
		if err != nil {
			logger.Error("failed to update interface status to ready", "interface", iface, "err", err)

			continue
		}
	}
}

func deleteInterfaces(vmNodes map[uint64]string) {
	logger.Debug("Configuring interfaces in worker")

	interfaces, err := db.GetInterfacesWithStatus(string(InterfaceStatusPreDeleting))
	if err != nil {
		logger.Error("failed to get interfaces with 'pre-creating' status", "error", err)

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
			logger.Error("failed to remove network interface from Proxmox VM", "error", err)

			continue
		}

		err = db.UpdateInterfaceStatus(iface.ID, string(InterfaceStatusDeleting))
		if err != nil {
			logger.Error("failed to update interface status to deleting", "interface", iface, "err", err)

			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			continue
		}

		if !isSuccessful {
			logger.Error("failed to remove network interface from Proxmox VM")

			continue
		}

		err = db.DeleteInterfaceByID(iface.ID)
		if err != nil {
			logger.Error("failed to delete interface from DB", "interface", iface, "err", err)

			continue
		}
	}
}

func configureInterfaces(vmNodes map[uint64]string) {
	logger.Debug("Configuring interfaces in worker")

	interfaces, err := db.GetInterfacesWithStatus(string(InterfaceStatusPreConfiguring))
	if err != nil {
		logger.Error("failed to get interfaces with 'pre-creating' status", "error", err)

		return
	}

	for _, iface := range interfaces {
		dbVM, err := db.GetVMByID(uint64(iface.VMID))
		if err != nil {
			logger.Error("failed to get VM by ID for interface", "interface_id", iface.ID, "vmid", iface.VMID, "err", err)

			continue
		}

		err = db.UpdateInterfaceStatus(iface.ID, string(InterfaceStatusConfiguring))
		if err != nil {
			logger.Error("failed to update interface status to configuring", "interface", iface, "err", err)

			continue
		}

		if !slices.Contains(goodVMStatesForInterfacesManipulation, VMStatus(dbVM.Status)) {
			logger.Warn("Can't configure interface. VM not in a good state for interface manipulation", "vmid", iface.VMID, "interface_id", iface.ID, "vm_status", dbVM.Status)

			continue
		}

		dbNet, err := db.GetNetByID(iface.VNetID)
		if err != nil {
			logger.Error("failed to get net by ID", "interface_id", iface.ID, "net_id", iface.VNetID, "err", err)

			continue
		}

		nodeName, ok := vmNodes[uint64(iface.VMID)]
		if !ok {
			logger.Error("Can't configure interface. VM not found on cluster resources", "vmid", iface.VMID, "interface_id", iface.ID)

			continue
		}

		node, err := getProxmoxNode(client, nodeName)
		if err != nil {
			logger.Error("Can't get node. Can't configure interface", "err", err, "vmid", iface.VMID, "interface_id", iface.ID)

			continue
		}

		vm, err := getProxmoxVM(node, int(iface.VMID))
		if err != nil {
			logger.Error("Can't get VM. Can't configure VM", "err", err, "vmid", iface.VMID)

			continue
		}

		mnets := vm.VirtualMachineConfig.Nets
		// We just check that a network exists for the local_id of the interface
		s := fmt.Sprintf("net%d", iface.LocalID)

		pnet, ok := mnets[s]
		if !ok {
			logger.Error("Can't configure interface. Network not found on Proxmox VM", "interface_id", iface.ID, "vmid", iface.VMID, "local_id", iface.LocalID)

			continue
		}

		if dbNet.VlanAware {
			pnet = substituteVlanTag(pnet, iface.VlanTag)
		}

		o := gprox.VirtualMachineOption{
			Name:  s,
			Value: pnet,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t, err := vm.Config(ctx, o)

		cancel()

		if err != nil {
			logger.Error("failed to add network interface to Proxmox VM", "error", err)

			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			continue
		}

		if !isSuccessful {
			logger.Error("failed to add network interface to Proxmox VM")

			continue
		}

		o2 := gprox.VirtualMachineOption{
			Name:  fmt.Sprintf("ipconfig%d", iface.LocalID),
			Value: "ip=" + iface.IPAdd,
		}

		if iface.Gateway != "" {
			o2.Value = fmt.Sprintf("%s,gw=%s", o2.Value, iface.Gateway)
		}

		logger.Debug("Configuring network interface on Proxmox VM", "option", o2)

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err = vm.Config(ctx, o2)

		cancel()

		if err != nil {
			logger.Error("failed to configure network interface on Proxmox VM", "error", err)

			continue
		}

		isSuccessful, err = waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			continue
		}

		if !isSuccessful {
			logger.Error("failed to configure network interface on Proxmox VM")

			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		err = vm.RegenerateCloudInitImage(ctx)

		cancel()

		if err != nil {
			logger.Error("failed to regenerate cloud-init image on Proxmox VM", "error", err)
		}

		err = db.UpdateInterfaceStatus(iface.ID, string(InterfaceStatusReady))
		if err != nil {
			logger.Error("failed to update interface status to ready", "interface", iface, "err", err)

			continue
		}
	}
}

func deleteBackups(mapVMContent map[uint64]string) {
	logger.Debug("Deleting backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeDelete)
	if err != nil {
		logger.Error("failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeDelete, "error", err)

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
			logger.Error("failed to get Proxmox node", "node", nodeName, "error", err)

			continue
		}

		storage, err := getProxmoxStorage(node, cBackup.Storage)
		if err != nil {
			logger.Error("failed to get Proxmox storage", "storage", cBackup.Storage, "error", err)

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
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.Error("failed to update backup request status to failed", "id", r.ID, "error", err)
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
			logger.Error("failed to update backup request status", "status", status, "id", r.ID, "error", err)
		}
	}
}

func restoreBackups(mapVMContent map[uint64]string) {
	logger.Debug("Restoring backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeRestore)
	if err != nil {
		logger.Error("failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeRestore, "error", err)

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
			logger.Error("failed to get Proxmox node", "node", nodeName, "error", err)

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
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.Error("failed to update backup request status to failed", "id", r.ID, "error", err)
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
			logger.Error("failed to update backup request status", "status", status, "id", r.ID, "error", err)
		}
	}
}

func createBackups(mapVMContent map[uint64]string) {
	logger.Debug("Creating backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeCreate)
	if err != nil {
		logger.Error("failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeCreate, "error", err)

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
			logger.Error("failed to get Proxmox node", "node", nodeName, "error", err)

			continue
		}

		notes, err := generateBackNotes(r.Name, r.Notes, r.OwnerID, r.OwnerType)
		if err != nil {
			logger.Error("failed to generate backup notes", "error", err)

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
			logger.Error("failed to wait for Proxmox task completion", "error", err)

			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.Error("failed to update backup request status to failed", "id", r.ID, "error", err)
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
			logger.Error("failed to update backup request status", "status", status, "id", r.ID, "error", err)
		}
	}
}

func configureSSHKeys(vmNodes map[uint64]string) {
	logger.Debug("Configuring SSH keys in worker")

	states := []string{string(VMStatusStopped), string(VMStatusRunning), string(VMStatusPaused)}

	ssht := db.GetLastSSHKeyUpdate()

	vmt, err := db.GetTimeOfLastCreatedVMWithStates(states)
	if err != nil {
		logger.Error("failed to get time of last created VM with states", "error", err)

		return
	}

	groupt := db.GetLastUserGroupUpdate()

	// Every 6 hours we force a reconfiguration of SSH keys
	if !lastConfigureSSHKeysTime.Before(time.Now().Add(-6*time.Hour)) &&
		lastConfigureSSHKeysTime.After(ssht) &&
		lastConfigureSSHKeysTime.After(vmt) &&
		lastConfigureSSHKeysTime.After(groupt) {
		logger.Debug("No need to configure SSH keys. No new SSH keys or VMs")

		return
	}

	// TODO: We could optimize this further by checking why the ssh keys table
	// changed and only updating the VMs of the users that have changes (unless
	// global keys changed)

	vms, err := db.GetVMsWithStates(states)
	if err != nil {
		logger.Error("failed to get VMs with 'stopped' status", "error", err)

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
			logger.Error("failed to get SSH keys for user", "vmid", v.ID, "ownerID", v.OwnerID, "ownerType", v.OwnerType, "err", err)

			continue
		}

		if v.IncludeGlobalSSHKeys {
			globalKeys, err := db.GetGlobalSSHKeys()
			if err != nil {
				logger.Error("failed to get global SSH keys", "vmid", v.ID, "err ", err)

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
			logger.Error("failed to set ssh keys on VM", "vmid", v.ID, "err", err)

			return
		}

		logger.Debug("Task finished", "isSuccessful", isSuccessful)

		if !isSuccessful {
			logger.Error("failed to set ssh keys on VM", "vmid", v.ID)

			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = vm.RegenerateCloudInitImage(ctx)

		cancel()

		if err != nil {
			logger.Error("failed to regenerate cloud init image on VM", "vmid", v.ID, "err", err)
		}

		if v.OwnerType == "Group" && lastConfigureSSHKeysTime.After(v.CreatedAt) {
			err = notify.SendSSHKeysChangedOnVMToGroup(v.OwnerID, v.Name)
			if err != nil {
				logger.Error("failed to send SSH keys changed notification to group", "vmid", v.ID, "err", err)
			}
		}
	}

	lastConfigureSSHKeysTime = time.Now()
}

func enforceVMLifetimes() {
	t := time.Now().AddDate(0, 3, 0) // 3 months from now

	vms, err := db.GetVMsWithLifetimesLessThanAndStatusIN(t, []string{
		string(VMStatusRunning),
		string(VMStatusStopped),
		string(VMStatusPaused),
	})
	if err != nil {
		logger.Error("failed to get VMs with lifetimes less than", "time", t, "error", err)

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
			logger.Error("failed to get VM expiration notifications for VM", "vmid", v.ID, "error", err)

			continue
		}

		switch {
		case v.LifeTime.Before(time.Now().Add(-7 * 24 * time.Hour)):
			// The VM expired more than 7 days ago, we delete it
			err := deleteVMBypass(v.ID)
			if err != nil {
				logger.Error("failed to delete expired VM", "vmid", v.ID, "error", err)

				continue
			}

			if v.OwnerType == "Group" {
				err = notify.SendVMEliminatedNotificationToGroup(v.OwnerID, v.Name)
			} else {
				err = notify.SendVMEliminatedNotification(v.OwnerID, v.Name)
			}

			if err != nil {
				logger.Error("failed to send VM eliminated notification", "vmid", v.ID, "error", err)
			}
		case v.LifeTime.Before(time.Now()) && !slices.ContainsFunc(notifications, fn(0)):
			if v.Status == string(VMStatusStopped) {
				if v.OwnerType == "Group" {
					err = notify.SendLifetimeOfVMExpiredToGroup(v.OwnerID, v.Name)
				} else {
					err = notify.SendLifetimeOfVMExpired(v.OwnerID, v.Name)
				}

				if err != nil {
					logger.Error("failed to send VM stopped notification", "vmid", v.ID, "error", err)
				}

				// We use the 0 days_before to indicate that the VM has expired
				_, err = db.NewVMExpirationNotification(v.ID, 0)
				if err != nil {
					logger.Error("failed to create VM expiration notification", "vmid", v.ID, "days_before", 0, "error", err)
				}

				continue
			}

			lastTimeFailed, exists := vmFailedChangeStatus[v.ID]
			if exists && lastTimeFailed.After(time.Now().Add(-30*time.Hour)) {
				// We failed to stop the VM less than an hour ago, we skip it
				continue
			}

			// The VM expired, but less than 7 days ago, we send the last notification
			// and stop the VM if it is running
			if v.Status != string(VMStatusStopped) {
				err := changeVMStatusBypass(v.ID, "stop")
				if err != nil {
					vmFailedChangeStatus[v.ID] = time.Now()
					logger.Error("failed to stop expired VM", "vmid", v.ID, "error", err)

					continue
				}
			}

			if v.OwnerType == "Group" {
				err = notify.SendVMStoppedNotificationToGroup(v.OwnerID, v.Name)
			} else {
				err = notify.SendVMStoppedNotification(v.OwnerID, v.Name)
			}

			if err != nil {
				logger.Error("failed to send VM stopped notification", "vmid", v.ID, "error", err)
			}

			// We use the 0 days_before to indicate that the VM has expired
			_, err = db.NewVMExpirationNotification(v.ID, 0)
			if err != nil {
				logger.Error("failed to create VM expiration notification", "vmid", v.ID, "days_before", 0, "error", err)
			}
		default:
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
						logger.Error("failed to send VM expiration notification", "vmid", v.ID, "days_before", i, "error", err)

						continue
					}

					_, err = db.NewVMExpirationNotification(v.ID, uint(i))
					if err != nil {
						logger.Error("failed to create VM expiration notification", "vmid", v.ID, "days_before", i, "error", err)
					}

					break
				}
			}
		}
	}
}
