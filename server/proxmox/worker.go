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

	gprox "github.com/luthermonson/go-proxmox"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func Worker() {
	time.Sleep(10 * time.Second)
	logger.Info("Starting Proxmox worker")

	for {
		// For all VMs we must check the status and take the necessary actions
		if !isProxmoxReachable {
			time.Sleep(20 * time.Second)
			continue
		}

		createVNets()
		deleteVNets()

		deleteVMs()
		createVMs()
		configureVMs()

		updateVMs()

		createInterfaces()
		deleteInterfaces()

		deleteBackups()
		restoreBackups()
		createBackups()

		time.Sleep(10 * time.Second)
	}
}

func createVNets() {
	logger.Debug("Creating VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VMStatusPreCreating))
	if err != nil {
		logger.With("error", err).Error("Failed to get VNets with 'pre-creating' status")
		return
	}

	cluster, err := getProxmoxCluster(client)
	if err != nil {
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
			logger.With("vnet", v.Name, "error", err).Error("Failed to create VNet in Proxmox")
			continue
		}

		err = db.UpdateVNetStatus(v.ID, string(VNetStatusCreating))
		if err != nil {
			logger.With("vnet", v.Name, "new_status", VNetStatusCreating, "err", err).Error("Failed to update status of VNet")
			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to apply SDN changes in Proxmox")
		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if err != nil {
		logger.With("error", err).Error("Failed to wait for Proxmox task completion")
		return
	}

	if !isSuccessful {
		logger.Error("Failed to apply SDN changes in Proxmox")
		// Set all VNets status to 'unknown'
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.With("vnet", v.Name, "new_status", VNetStatusUnknown, "err", err).Error("Failed to update status of VNet")
			}
		}
		return
	} else {
		logger.Info("SDN changes applied successfully")
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusReady))
			if err != nil {
				logger.With("vnet", v.Name, "new_status", VNetStatusReady, "err", err).Error("Failed to update status of VNet")
			}
		}
	}
}

func deleteVNets() {
	logger.Debug("Deleting VNets in worker")

	vnets, err := db.GetVNetsWithStatus(string(VNetStatusPreDeleting))
	if err != nil {
		logger.With("error", err).Error("Failed to get VNets with 'pre-deleting' status")
		return
	}

	if len(vnets) == 0 {
		return
	}

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		return
	}

	for _, v := range vnets {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := cluster.DeleteSDNVNet(ctx, v.Name)
		cancel()
		if err != nil {
			logger.With("vnet", v.Name, "error", err).Error("Failed to delete VNet from Proxmox")
			continue
		}

		err = db.UpdateVNetStatus(v.ID, string(VNetStatusDeleting))
		if err != nil {
			logger.With("vnet", v.Name, "new_status", VNetStatusDeleting, "err", err).Error("Failed to update status of VNet")
			continue
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to apply SDN changes in Proxmox")
		return
	}

	isSuccessful, err := waitForProxmoxTaskCompletion(task)
	if isSuccessful {
		logger.Info("SDN changes applied successfully")
		for _, v := range vnets {
			err = db.DeleteNetByID(v.ID)
			if err != nil {
				logger.With("vnet", v.Name, "err", err).Error("Failed to delete VNet from DB")
			}
		}
	} else {
		logger.Error("Failed to apply SDN changes in Proxmox")
		for _, v := range vnets {
			err = db.UpdateVNetStatus(v.ID, string(VNetStatusUnknown))
			if err != nil {
				logger.With("vnet", v.Name, "new_status", VNetStatusUnknown, "err", err).Error("Failed to update status of VNet")
			}
		}
	}
}

// createVMs creates VMs from proxmox that are in the 'pre-creating' status.
func createVMs() {
	logger.Debug("Creating VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreCreating))
	if err != nil {
		logger.With("error", err).Error("Failed to get VMs with 'creating' status")
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

	cloningOptions := gprox.VirtualMachineCloneOptions{
		Full:   optionFull,
		Target: cClone.TargetNode,
		Name:   "sasso-001", // TODO: Find a meaningful name
	}

	for _, v := range vms {
		if v.Status != string(VMStatusPreCreating) {
			continue
		}
		logger.Info("Cloning VM", "vmid", v.ID)
		// Create the VM in Proxmox
		// Creation implies cloning a template
		cloningOptions.NewID = int(v.ID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, task, err := templateVm.Clone(ctx, &cloningOptions)
		cancel()
		if err != nil {
			logger.With("vmid", v.ID, "error", err).Error("Failed to clone VM")
			continue
		}
		err = db.UpdateVMStatus(v.ID, string(VMStatusCreating))
		if err != nil {
			logger.With("vmid", v.ID, "new_status", VMStatusCreating, "err", err).Error("Failed to update status of VM")
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(task)
		if isSuccessful {
			err = db.UpdateVMStatus(v.ID, string(VMStatusPreConfiguring))
			if err != nil {
				logger.With("vmid", v.ID, "new_status", VMStatusStopped, "err", err).Error("Failed to update status of VM")
			}
		} else {
			// We could set the status as pre-creating to trigger a recreation, but
			// for now we just set it to unknown
			err = db.UpdateVMStatus(v.ID, string(VMStatusUnknown))
			if err != nil {
				logger.With("vmid", v.ID, "new_status", VMStatusUnknown, "err", err).Error("Failed to update status of VM")
			}
		}
	}
}

// deleteVMs deletes VMs from proxmox that are in the 'pre-deleting' status.
func deleteVMs() {
	logger.Debug("Deleting VMs in worker")

	vms, err := db.GetVMsWithStatus(string(VMStatusPreDeleting))
	if err != nil {
		logger.With("error", err).Error("Failed to get VMs with 'deleting' status")
		return
	}

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		return
	}

	VMLocation, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		return
	}

	for _, v := range vms {
		logger.With("vmid", v.ID).Info("Deleting VM")

		err := db.DeleteAllInterfacesByVMID(v.ID)
		if err != nil {
			logger.With("vmid", v.ID, "err", err).Error("Failed to delete interfaces for VM")
		}

		nodeName, ok := VMLocation[v.ID]
		if !ok {
			logger.With("vmid", v.ID).Error("Can't delete VM. Not found on cluster resources")

			// If the VM is not found on Proxmox, we just delete it from the DB
			err = db.DeleteVMByID(v.ID)
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to delete VM")
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
				logger.With("err", err, "vmid", v.ID).Error("Can't stop VM before deletion")
				continue
			}
			isSuccessful, err := waitForProxmoxTaskCompletion(task)
			if err != nil {
				logger.With("err", err, "vmid", v.ID).Error("Can't wait for stop VM task completion")
				continue
			}
			if !isSuccessful {
				logger.With("vmid", v.ID).Error("Can't stop VM before deletion")
				continue
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		task, err := vm.Delete(ctx)
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", v.ID).Error("Can't delete VM")
			continue
		}
		err = db.UpdateVMStatus(v.ID, string(VMStatusDeleting))
		if err != nil {
			logger.With("vmid", v.ID, "new_status", VMStatusDeleting, "err", err).Warn("Failed to update status of VM")
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(task)
		cancel()
		if isSuccessful {
			err = db.DeleteVMByID(v.ID)
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to delete VM")
			}
		} else {
			// We could set the status as pre-creating to trigger a recreation, but
			// for now we just set it to unknown
			err = db.UpdateVMStatus(v.ID, string(VMStatusUnknown))
			if err != nil {
				logger.With("vmid", v.ID, "new_status", VMStatusUnknown, "err", err).Error("Failed to update status of VM")
			}
		}
	}
}

// This function configures VMs that are in the 'pre-configuring' status.
// Configuration includes setting the number of cores, RAM and disk size
func configureVMs() {
	logger.Debug("Configuring VMs in worker")

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		return
	}

	vmNodes, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		return
	}

	vms, err := db.GetVMsWithStatus(string(VMStatusPreConfiguring))
	if err != nil {
		logger.With("error", err).Error("Failed to get VMs with 'pre-configuring' status")
		return
	}

	for _, v := range vms {
		nodeName, ok := vmNodes[v.ID]
		if !ok {
			logger.With("vmid", v.ID).Error("Can't configure VM. Not found on cluster resources")
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
		logger.With("vmid", v.ID).Info("Configuring VM")

		if vm.VirtualMachineConfig.Cores != int(v.Cores) {
			coresOption := gprox.VirtualMachineOption{
				Name:  "cores",
				Value: v.Cores,
			}
			isSuccessful, err := configureVM(vm, coresOption)
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set cores on VM")
				return
			}
			logger.With("isSuccessful", isSuccessful).Info("Task finished")
			if !isSuccessful {
				logger.With("vmid", v.ID).Error("Failed to set cores on VM")
			}
		}

		if uint(vm.VirtualMachineConfig.Memory) != v.RAM {
			ramOption := gprox.VirtualMachineOption{
				Name:  "memory",
				Value: v.RAM,
			}
			isSuccessful, err := configureVM(vm, ramOption)
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set ram on VM")
				continue
			}
			logger.With("isSuccessful", isSuccessful).Info("Task finished")
			if !isSuccessful {
				logger.With("vmid", v.ID).Error("Failed to set ram on VM")
			}
		}

		scsi0, ok := vm.VirtualMachineConfig.SCSIs["scsi0"]
		if !ok {
			logger.With("vmid", v.ID).Error("Failed to find SCSI0 on VM")
			continue
		}
		st, err := parseStorageFromString(scsi0)
		if err != nil {
			logger.With("vmid", v.ID, "scsi0", scsi0).Error("Failed to parse storage on SCSI0")
			continue
		}

		if st.Size < uint(v.Disk) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			diff := uint(v.Disk) - st.Size
			t, err := vm.ResizeDisk(ctx, "scsi0", fmt.Sprintf("+%dG", diff))
			cancel()
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set resize disk on VM")
				continue
			}

			isSuccessful, err := waitForProxmoxTaskCompletion(t)
			logger.With("isSuccessful", isSuccessful).Info("Task finished")
			if !isSuccessful {
				logger.With("vmid", v.ID).Error("Failed to resize disk on VM")
			}
		}

		sshKeys, err := db.GetSSHKeysByUserID(v.UserID)
		if err != nil {
			logger.With("vmid", v.ID, "userid", v.UserID, "err", err).Error("Failed to get SSH keys for user")
			continue
		}

		if v.IncludeGlobalSSHKeys {
			globalKeys, err := db.GetGlobalSSHKeys()
			if err != nil {
				logger.With("vmid", v.ID, "userid", v.UserID, "err ", err).Error("Failed to get global SSH keys")
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

		if vm.VirtualMachineConfig.SSHKeys != cloudInitKeys {
			sshOption := gprox.VirtualMachineOption{
				Name:  "sshkeys",
				Value: cloudInitKeys,
			}
			isSuccessful, err := configureVM(vm, sshOption)
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set ssh keys on VM")
				return
			}
			logger.With("isSuccessful", isSuccessful).Info("Task finished")
			if !isSuccessful {
				logger.With("vmid", v.ID).Error("Failed to set ssh keys on VM")
				continue
			}
		}

		err = db.UpdateVMStatus(v.ID, string(VMStatusStopped))
		if err != nil {
			logger.With("vmid", v.ID, "new_status", VMStatusStopped, "err", err).Error("Failed to update status of VM")
		}

		logger.With("vm", vm, "vm.VirtualMachineConfig", vm.VirtualMachineConfig).Info("VM configured")
	}
}

// updateVMs updates the status of VMs in the database based on their current status in Proxmox.
func updateVMs() {
	logger.Debug("Updating VMs in worker")

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		return
	}

	resources, err := getProxmoxResources(cluster, "vm")
	allVMStatus := []string{string(VMStatusRunning), string(VMStatusStopped), string(VMStatusSuspended)}

	activeVMs, err := db.GetAllActiveVMs()
	if err != nil {
		logger.With("err", err).Error("Can't get active VMs from DB")
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

		if !slices.Contains(allVMStatus, r.Status) {
			logger.With("vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status).Error("VM status unrecognised, setting status to unknown")

			err := db.UpdateVMStatus(r.VMID, string(VMStatusUnknown))
			if err != nil {
				logger.With("vmid", r.VMID, "new_status", VMStatusDeleting, "err", err).Error("Failed to update status of VM")
			}
		} else if r.Status != vm.Status {
			logger.With("vmid", r.VMID, "new_status", r.Status, "old_status", vm.Status).Warn("VM changed status on proxmox unexpectedly")

			err := db.UpdateVMStatus(r.VMID, r.Status)
			if err != nil {
				logger.With("vmid", r.VMID, "new_status", r.Status, "err", err).Error("Failed to update status of VM")
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

		logger.With("vmid", vmid, "status", activeVMs[i].Status).Error("VM not found on proxmox but is on sasso. Setting status to unknown")

		err := db.UpdateVMStatus(vmid, string(VMStatusUnknown))
		if err != nil {
			logger.With("vmid", vmid, "new_status", VMStatusUnknown, "err", err).Error("Failed to update status of VM")
		}
	}
}

func createInterfaces() {
	logger.Debug("Configuring interfaces in worker")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.With("err", err).Error("Can't get cluster")
		return
	}

	vmNodes, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.With("err", err).Error("Can't map VMID to Proxmox nodes")
		return
	}

	interfaces, err := db.GetInterfacesWithStatus(string(InterfaceStatusPreCreating))
	if err != nil {
		logger.With("error", err).Error("Failed to get interfaces with 'pre-creating' status")
		return
	}

	for _, iface := range interfaces {
		nodeName, ok := vmNodes[uint64(iface.VMID)]
		if !ok {
			logger.With("vmid", iface.VMID, "interface_id", iface.ID).Error("Can't configure interface. VM not found on cluster resources")
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		node, err := client.Node(ctx, nodeName)
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", iface.VMID, "interface_id", iface.ID).Error("Can't get node. Can't configure interface")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		vm, err := node.VirtualMachine(ctx, int(iface.VMID))
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", iface.VMID).Error("Can't get VM. Can't configure VM")
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
		logger.With("mnets", mnets, "snet", snet).Debug("Current network interfaces on Proxmox VM")
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
			logger.With("interface_id", iface.ID, "net_id", iface.VNetID, "err", err).Error("Failed to get net by ID")
			continue
		}

		o := gprox.VirtualMachineOption{
			Name:  "net" + strconv.Itoa(firstEmptyIndex),
			Value: fmt.Sprintf("virtio,bridge=%s,firewall=1", vnet.Name),
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err := vm.Config(ctx, o)
		cancel()
		if err != nil {
			logger.With("error", err).Error("Failed to add network interface to Proxmox VM")
			continue
		}
		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
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

		logger.With("option", o2).Debug("Configuring network interface on Proxmox VM")

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err = vm.Config(ctx, o2)
		cancel()
		if err != nil {
			logger.With("error", err).Error("Failed to configure network interface on Proxmox VM")
			continue
		}
		isSuccessful, err = waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
			continue
		}
		if !isSuccessful {
			logger.Error("Failed to configure network interface on Proxmox VM")
			continue
		}

		iface.LocalID = uint(firstEmptyIndex)
		iface.Status = string(InterfaceStatusReady)
		err = db.UpdateInterface(&iface)
		if err != nil {
			logger.With("interface", iface, "err", err).Error("Failed to update interface status to ready")
			continue
		}
	}
}

func deleteInterfaces() {
	logger.Debug("Configuring interfaces in worker")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.With("err", err).Error("Can't get cluster")
		return
	}

	vmNodes, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.With("err", err).Error("Can't map VMID to Proxmox nodes")
		return
	}

	interfaces, err := db.GetInterfacesWithStatus(string(InterfaceStatusPreDeleting))
	if err != nil {
		logger.With("error", err).Error("Failed to get interfaces with 'pre-creating' status")
		return
	}

	for _, iface := range interfaces {
		nodeName, ok := vmNodes[uint64(iface.VMID)]
		if !ok {
			logger.With("vmid", iface.VMID, "interface_id", iface.ID).Error("Can't configure interface. VM not found on cluster resources")
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		node, err := client.Node(ctx, nodeName)
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", iface.VMID, "interface_id", iface.ID).Error("Can't get node. Can't configure interface")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		vm, err := node.VirtualMachine(ctx, int(iface.VMID))
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", iface.VMID).Error("Can't get VM. Can't configure VM")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		t, err := vm.Config(ctx, gprox.VirtualMachineOption{
			Name:  "delete",
			Value: fmt.Sprintf("net%d", iface.LocalID),
		})
		cancel()
		if err != nil {
			logger.With("error", err).Error("Failed to remove network interface from Proxmox VM")
			continue
		}

		err = db.UpdateInterfaceStatus(iface.ID, string(InterfaceStatusDeleting))
		if err != nil {
			logger.With("interface", iface, "err", err).Error("Failed to update interface status to deleting")
			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
			continue
		}
		if !isSuccessful {
			logger.Error("Failed to remove network interface from Proxmox VM")
			continue
		}

		err = db.DeleteInterfaceByID(iface.ID)
		if err != nil {
			logger.With("interface", iface, "err", err).Error("Failed to delete interface from DB")
			continue
		}
	}
}

func deleteBackups() {
	logger.Debug("Deleting backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeDelete)
	if err != nil {
		logger.With("error", err).Error("Failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeDelete)
		return
	}

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox cluster")
		return
	}

	mapVMContent, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.With("error", err).Error("Failed to map VMID to content")
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
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.With("error", err).Error("Failed to update backup request status to failed", "id", r.ID)
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
			logger.With("error", err).Error("Failed to update backup request status", "status", status, "id", r.ID)
		}
	}
}

func restoreBackups() {
	logger.Debug("Restoring backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeRestore)
	if err != nil {
		logger.With("error", err).Error("Failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeRestore)
		return
	}

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox cluster")
		return
	}

	mapVMContent, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.With("error", err).Error("Failed to map VMID to content")
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
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.With("error", err).Error("Failed to update backup request status to failed", "id", r.ID)
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
			logger.With("error", err).Error("Failed to update backup request status", "status", status, "id", r.ID)
		}
	}
}

func createBackups() {
	logger.Debug("Creating backups in worker")

	bkr, err := db.GetBackupRequestWithStatusAndType(BackupRequestStatusPending, BackupRequestTypeCreate)
	if err != nil {
		logger.With("error", err).Error("Failed to get backup requests", "status", BackupRequestStatusPending, "type", BackupRequestTypeCreate)
		return
	}

	cluster, err := getProxmoxCluster(client)
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox cluster")
		return
	}

	mapVMContent, err := mapVMIDToProxmoxNodes(cluster)
	if err != nil {
		logger.With("error", err).Error("Failed to map VMID to content")
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

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t, err := node.Vzdump(ctx, &gprox.VirtualMachineBackupOptions{
			Storage:       cBackup.Storage,
			VMID:          uint64(r.VMID),
			Mode:          "snapshot",
			Remove:        false,
			Compress:      "zstd",
			NotesTemplate: fmt.Sprintf("{{guestname}} %s", BackupNoteString),
		})
		cancel()
		if err != nil {
			slog.Error("failed to create vzdump", "error", err)
			continue
		}

		isSuccessful, err := waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
			err = db.UpdateBackupRequestStatus(r.ID, BackupRequestStatusFailed)
			if err != nil {
				logger.With("error", err).Error("Failed to update backup request status to failed", "id", r.ID)
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
			logger.With("error", err).Error("Failed to update backup request status", "status", status, "id", r.ID)
		}
	}
}
