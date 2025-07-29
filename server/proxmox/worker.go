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
	for {
		// For all VMs we must check the status and take the necessary actions
		createVMs()
		deleteVMs()

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
		db.UpdateVMStatus(v.ID, string(VMStatusCreating))

		ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
		status, completed, err := task.WaitForCompleteStatus(ctx, 120, 1)
		cancel()
		logger.With("status", status, "completed", completed).Info("Task finished")
	}
}

func deleteVMs() {
	vms, err := db.GetVMsWithStatus(string(VMStatusPreDeleting))
	if err != nil {
		logger.With("error", err).Error("Failed to get VMs with 'deleting' status")
		return
	}

	for _, v := range vms {
		if v.Status != string(VMStatusPreDeleting) {
			continue
		}
		logger.With("vmid", v.ID).Info("Deleting VM")
		// Delete the VM in Proxmox
		// Wait for its deletion
		// Repeat
	}
}
