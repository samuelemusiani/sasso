// Sasso will write to the DB what the current state should look like. The
// worker will read the DB and take take care of all the operations that needs
// to be done

package proxmox

import (
	"context"
	"time"

	"samuelemusiani/sasso/server/db"

	gprox "github.com/luthermonson/go-proxmox"
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

		deleteVMs()
		createVMs()

		time.Sleep(10 * time.Second)
	}
}

func createVMs() {
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
		logger.With("status", isSuccessful, "completed", completed).Info("Task finished")
		if completed {
			if isSuccessful {
				err = db.UpdateVMStatus(v.ID, string(VMStatusStopped))
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
}

func deleteVMs() {
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
		status, completed, err := task.WaitForCompleteStatus(ctx, 30, 1)
		cancel()
		logger.With("status", status, "completed", completed).Info("Task finished")
	}
}
