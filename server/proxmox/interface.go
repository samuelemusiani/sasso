package proxmox

import (
	"slices"

	"samuelemusiani/sasso/server/db"
)

type InterfaceStatus string

var (
	InterfaceStatusUnknown        InterfaceStatus = "unknown"
	InterfaceStatusPreCreating    InterfaceStatus = "pre-creating"
	InterfaceStatusCreating       InterfaceStatus = "creating"
	InterfaceStatusPreDeleting    InterfaceStatus = "pre-deleting"
	InterfaceStatusDeleting       InterfaceStatus = "deleting"
	InterfaceStatusReady          InterfaceStatus = "ready"
	InterfaceStatusPreConfiguring InterfaceStatus = "pre-configuring"
	InterfaceStatusConfiguring    InterfaceStatus = "configuring"
)

type Interface struct {
	ID      uint            `json:"id"`
	LocalID uint            `json:"-"` // Internal use only
	VNetID  uint            `json:"vnet_id"`
	VlanTag uint16          `json:"vlan_tag"`
	IPAdd   string          `json:"ip_add"`
	Gateway string          `json:"gateway"`
	Status  InterfaceStatus `json:"status"`
}

var goodVMStatesForInterfacesManipulation = []VMStatus{VMStatusRunning, VMStatusStopped, VMStatusSuspended, VMStatusPreConfiguring, VMStatusConfiguring}

func NewInterface(VMID uint, vnetID uint, vlanTag uint16, ipAdd string, gateway string) (*Interface, error) {
	vm, err := db.GetVMByID(uint64(VMID))
	if err != nil {
		logger.Error("Failed to get VM by ID", "VMID", VMID, "error", err)
		return nil, err
	}
	if !slices.Contains(goodVMStatesForInterfacesManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to add an interface", "VMID", VMID, "status", vm.Status)
		return nil, ErrInvalidVMState
	}

	iface, err := db.NewInterface(VMID, vnetID, vlanTag, ipAdd, gateway, string(InterfaceStatusPreCreating))
	if err != nil {
		logger.Error("Failed to create new interface", "error", err)
		return nil, err
	}
	return InterfaceFromDB(iface), nil
}

func InterfaceFromDB(dbIface *db.Interface) *Interface {
	return &Interface{
		ID:      dbIface.ID,
		LocalID: dbIface.ID,
		VNetID:  dbIface.VNetID,
		VlanTag: dbIface.VlanTag,
		IPAdd:   dbIface.IPAdd,
		Gateway: dbIface.Gateway,
		Status:  InterfaceStatus(dbIface.Status),
	}
}

func DeleteInterface(id uint) error {
	i, err := db.GetInterfaceByID(id)
	if err != nil {
		logger.Error("Failed to get interface by ID", "interfaceID", id, "error", err)
		return err
	}
	vm, err := db.GetVMByID(uint64(i.VMID))
	if err != nil {
		logger.Error("Failed to get VM by ID", "VMID", i.VMID, "error", err)
		return err
	}
	if !slices.Contains(goodVMStatesForInterfacesManipulation, VMStatus(vm.Status)) {
		logger.Error("VM is not in a valid state to add an interface", "VMID", i.VMID, "status", vm.Status)
		return ErrInvalidVMState
	}

	err = db.UpdateInterfaceStatus(id, string(InterfaceStatusPreDeleting))
	if err != nil {
		logger.Error("Failed to set interface status to pre-deleting", "interfaceID", id, "error", err)
		return err
	}
	return nil
}
