// Sasso will write to the DB what the current state should look like. The
// worker will read the DB and take take care of all the operations that needs
// to be done

package proxmox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox cluster")
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

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to apply SDN changes in Proxmox")
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
	isSuccessful, completed, err := task.WaitForCompleteStatus(ctx, 120, 1)
	cancel()
	logger.With("status", isSuccessful, "completed", completed).Info("SDN apply task finished")
	if !completed {
		logger.Error("SDN apply task did not complete")
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
	}

	client := &http.Client{Timeout: 10 * time.Second}

	logger.Info("SDN changes applied successfully")
	for _, v := range vnets {
		netRequest := struct {
			VNet   string `json:"vnet"`
			VNetID uint32 `json:"vnet_id"`
		}{
			VNet:   v.Name,
			VNetID: v.Tag,
		}

		netMarshal, err := json.Marshal(netRequest)
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to marshal net request")
			continue
		}

		req, err := http.NewRequest("POST", cGateway.Server+"/api/net", bytes.NewReader(netMarshal))
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to create net request")
			continue
		}

		req.Header.Set("Authorization", cGateway.Secret)
		res, err := client.Do(req)
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to send net request")
			continue
		}

		logger.With("vnet", v.Name, "status_code", res.StatusCode).Debug("Net request sent")

		ticketResponse := struct {
			TicketID string `json:"ticket_id"`
		}{}
		err = json.NewDecoder(res.Body).Decode(&ticketResponse)
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to decode net request response")
			continue
		}

		logger.With("vnet", v.Name, "ticket_id", ticketResponse.TicketID).Info("Net request ticket created")

		netResponse := struct {
			ID          uint   `json:"id"`           // ID of the request
			RequestType string `json:"request_type"` // Type of the request (e.g., "new_network", "delete_network")
			Request     struct {
				VNet   string `json:"vnet"`    // Name of the new VNet
				VNetID uint   `json:"vnet_id"` // ID of the new VNet (VXLAN ID)

				Status  string `json:"status"`  // Status of the request
				Success bool   `json:"success"` // True if the request was successful
				Error   string `json:"error"`   // Error message if the request failed

				Subnet    string `json:"Subnet"`    // Subnet of the new VNet
				RouterIP  string `json:"router_ip"` // Router IP of the new VNet
				Broadcast string `json:"broadcast"` // Broadcast address of the new VNet
			} `json:"request"`
		}{}

		for {
			req, err := http.NewRequest("GET", cGateway.Server+"/api/ticket/"+ticketResponse.TicketID, nil)
			if err != nil {
				logger.With("vnet", v.Name, "err", err).Error("Failed to create ticket status request")
				break
			}

			req.Header.Set("Authorization", cGateway.Secret)
			res, err := client.Do(req)
			if err != nil {
				logger.With("vnet", v.Name, "err", err).Error("Failed to send ticket status request")
				break
			}

			err = json.NewDecoder(res.Body).Decode(&netResponse)
			if err != nil {
				logger.With("vnet", v.Name, "err", err).Error("Failed to decode ticket status response")
				break
			}
			logger.With("vnet", v.Name, "net_response", netResponse).Debug("Ticket status response")

			if netResponse.Request.Status == "completed" {
				break
			}

			time.Sleep(1 * time.Second)
		}

		if netResponse.Request.Status != "completed" {
			continue
		}

		net, err := db.GetNetByID(v.ID)
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to get net by ID")
			continue
		}

		net.Status = string(VNetStatusReady)
		net.Subnet = netResponse.Request.Subnet
		net.Gateway = netResponse.Request.RouterIP
		net.Broadcast = netResponse.Request.Broadcast

		err = db.UpdateVNet(net)
		if err != nil {
			logger.With("vnet", net).Error("Failed to update status of VNet")
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox cluster")
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

	client := &http.Client{Timeout: 10 * time.Second}

	logger.Info("SDN changes applied successfully")
	for _, v := range vnets {
		netRequest := struct {
			VNet   string `json:"vnet"`
			VNetID uint32 `json:"vnet_id"`
		}{
			VNet:   v.Name,
			VNetID: v.Tag,
		}

		netMarshal, err := json.Marshal(netRequest)
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to marshal net request")
			continue
		}

		req, err := http.NewRequest("DELETE", cGateway.Server+"/api/net", bytes.NewReader(netMarshal))
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to create net request")
			continue
		}

		req.Header.Set("Authorization", cGateway.Secret)
		res, err := client.Do(req)
		if err != nil {
			logger.With("vnet", v.Name, "err", err).Error("Failed to send net request")
			continue
		}

		logger.With("vnet", v.Name, "status_code", res.StatusCode).Debug("Net request sent")
	}

	time.Sleep(10 * time.Second) //TODO: We should wait for the last ticket to finish

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	task, err := cluster.SDNApply(ctx)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to apply SDN changes in Proxmox")
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
	isSuccessful, completed, err := task.WaitForCompleteStatus(ctx, 120, 1)
	cancel()
	logger.With("status", isSuccessful, "completed", completed).Info("SDN apply task finished")
	if !completed {
		logger.Error("SDN apply task did not complete")
		return
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	node, err := client.Node(ctx, cTemplate.Node)
	cancel()

	if err != nil {
		logger.With("node", cTemplate.Node, "error", err).Error("Failed to get Proxmox node")
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	templateVm, err := node.VirtualMachine(ctx, cTemplate.VMID)
	cancel()

	if err != nil {
		logger.With("node", node.Name, "vmid", cTemplate.VMID, "error", err).Error("Failed to get template VM")
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
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
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

		ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
		isSuccessful, completed, err := task.WaitForCompleteStatus(ctx, 120, 1)
		cancel()
		logger.With("status", isSuccessful, "completed", completed).Info("VM Clone task finished")
		if !completed {
			logger.Error("VM Clone task did not complete")
			return
		}
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := client.Cluster(ctx)
	cancel()

	if err != nil {
		logger.With("err", err).Error("Can't get cluster")
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
	resources, err := cluster.Resources(ctx, "vm")
	cancel()

	if err != nil {
		logger.With("err", err).Error("Can't get cluster resources")
		return
	}

	VMLocation := make(map[uint64]string)
	for i := range resources {
		if resources[i].Type != "qemu" {
			continue
		}
		VMLocation[resources[i].VMID] = resources[i].Node
	}

	for _, v := range vms {
		if v.Status != string(VMStatusPreDeleting) {
			continue
		}
		logger.With("vmid", v.ID).Info("Deleting VM")

		nodeName, ok := VMLocation[v.ID]
		if !ok {
			logger.With("vmid", v.ID).Error("Can't delete VM. Not found on cluster resources")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		node, err := client.Node(ctx, nodeName)
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", v.ID).Error("Can't get node. Can't delete VM")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		vm, err := node.VirtualMachine(ctx, int(v.ID))
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", v.ID).Error("Can't get VM. Can't delete VM")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		task, err := vm.Delete(ctx)
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", v.ID).Error("Can't delete VM")
			continue
		}
		err = db.UpdateVMStatus(v.ID, string(VMStatusDeleting))
		if err != nil {
			logger.With("vmid", v.ID, "new_status", VMStatusDeleting, "err", err).Error("Failed to update status of VM")
		}

		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		isSuccessful, completed, err := task.WaitForCompleteStatus(ctx, 30, 1)
		cancel()
		logger.With("isSuccessful", isSuccessful, "completed", completed).Info("Task finished")

		if completed {
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
}

// This function configures VMs that are in the 'pre-configuring' status.
// Configuration includes setting the number of cores, RAM and disk size
func configureVMs() {
	logger.Debug("Configuring VMs in worker")

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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		node, err := client.Node(ctx, nodeName)
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", v.ID).Error("Can't get node. Can't configure VM")
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		vm, err := node.VirtualMachine(ctx, int(v.ID))
		cancel()
		if err != nil {
			logger.With("err", err, "vmid", v.ID).Error("Can't get VM. Can't configure VM")
			continue
		}
		logger.With("vmid", v.ID).Info("Configuring VM")

		if vm.VirtualMachineConfig.Cores != int(v.Cores) {
			coresOption := gprox.VirtualMachineOption{
				Name:  "cores",
				Value: v.Cores,
			}

			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			t, err := vm.Config(ctx, coresOption)
			cancel()
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set cores on VM")
				continue
			}
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			isSuccessful, completed, err := t.WaitForCompleteStatus(ctx, 30, 1)
			cancel()
			logger.With("isSuccessful", isSuccessful, "completed", completed).Info("Task finished")
			if !completed || !isSuccessful {
				logger.With("vmid", v.ID).Error("Failed to set cores on VM")
			}
		}

		if uint(vm.VirtualMachineConfig.Memory) != v.RAM {
			ramOption := gprox.VirtualMachineOption{
				Name:  "memory",
				Value: v.RAM,
			}

			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			t, err := vm.Config(ctx, ramOption)
			cancel()
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set ram on VM")
				continue
			}
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			isSuccessful, completed, err := t.WaitForCompleteStatus(ctx, 30, 1)
			cancel()
			logger.With("isSuccessful", isSuccessful, "completed", completed).Info("Task finished")
			if !completed || !isSuccessful {
				logger.With("vmid", v.ID).Error("Failed to set ram on VM")
			}
		}

		st, err := parseStorageFromString(vm.VirtualMachineConfig.SCSI0)
		if err != nil {
			logger.With("vmid", v.ID, "scsi0", vm.VirtualMachineConfig.SCSI0).Error("Failed to parse storage on SCSI0")
			continue
		}

		if st.Size < uint(v.Disk) {
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			diff := uint(v.Disk) - st.Size
			t, err := vm.ResizeDisk(ctx, "scsi0", fmt.Sprintf("+%dG", diff))
			cancel()
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set resize disk on VM")
				continue
			}

			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			isSuccessful, completed, err := t.WaitForCompleteStatus(ctx, 30, 1)
			cancel()
			logger.With("isSuccessful", isSuccessful, "completed", completed).Info("Task finished")
			if !completed || !isSuccessful {
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

		sshOption := gprox.VirtualMachineOption{
			Name:  "sshkeys",
			Value: cloudInitKeys,
		}

		if vm.VirtualMachineConfig.SSHKeys != cloudInitKeys {
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			t, err := vm.Config(ctx, sshOption)
			cancel()
			if err != nil {
				logger.With("vmid", v.ID, "err", err).Error("Failed to set ssh keys on VM")
				continue
			}

			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			isSuccessful, completed, err := t.WaitForCompleteStatus(ctx, 30, 1)
			cancel()
			logger.With("isSuccessful", isSuccessful, "completed", completed).Info("Task finished")
			if !completed || !isSuccessful {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.With("err", err).Error("Can't get cluster")
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
	resources, err := cluster.Resources(ctx, "vm")
	cancel()

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
		} else {
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
		// This is a temporary workaround
		mnets := vm.VirtualMachineConfig.MergeNets()
		var snet = make([]string, len(mnets))
		var i int = 0
		for k := range mnets {
			snet[i] = k
			i++
		}
		slices.Sort(snet)
		logger.With("mnets", mnets, "snet", snet).Debug("Current network interfaces on Proxmox VM")
		var firstEmptyIndex int = -1
		for i := range snet {
			if !strings.HasSuffix(snet[i], strconv.Itoa(i)) {
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
		_, _, err = waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
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
		_, _, err = waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
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

		_, _, err = waitForProxmoxTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
			continue
		}

		err = db.DeleteInterfaceByID(iface.ID)
		if err != nil {
			logger.With("interface", iface, "err", err).Error("Failed to delete interface from DB")
			continue
		}
	}
}
