// Sasso will write to the DB what the current state should look like. The
// worker will read the DB and take take care of all the operations that needs
// to be done

package proxmox

import (
	"samuelemusiani/sasso/server/db"
	"time"
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
	vms, err := db.GetVMsWithStatus(string(VMStatusPreCreation))
	if err != nil {
		logger.Error("Failed to get VMs with 'creating' status", "error", err)
		return
	}

	for _, v := range vms {
		logger.Info("Starting VM", "vmid", v.ID)
		// Create the VM in Proxmox
		// Wait for its creation
		// Repeat
	}
}

func deleteVMs() {
	vms, err := db.GetVMsWithStatus(string(VMStatusPreDeletion))
	if err != nil {
		logger.Error("Failed to get VMs with 'deleting' status", "error", err)
		return
	}

	for _, v := range vms {
		logger.Info("Deleting VM", "vmid", v.ID)
		// Delete the VM in Proxmox
		// Wait for its deletion
		// Repeat
	}
}
