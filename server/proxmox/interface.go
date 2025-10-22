package proxmox

import (
	"errors"
	"slices"

	"samuelemusiani/sasso/server/db"

	"github.com/seancfoley/ipaddress-go/ipaddr"
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

	ErrInterfaceNotFound      = errors.New("interface not found")
	ErrInvalidInterfaceConfig = errors.New("invalid interface configuration")
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
		if err == db.ErrNotFound {
			return ErrInterfaceNotFound
		}
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

func UpdateInterface(iface *Interface) error {
	dbIface, err := db.GetInterfaceByID(iface.ID)
	if err != nil {
		if err == db.ErrNotFound {
			return ErrInterfaceNotFound
		}
		logger.Error("Failed to get interface by ID", "interfaceID", iface.ID, "error", err)
		return err
	}

	dbIface.VNetID = iface.VNetID
	dbIface.VlanTag = iface.VlanTag
	dbIface.IPAdd = iface.IPAdd
	dbIface.Gateway = iface.Gateway
	// Status is not updated here

	err = db.UpdateInterface(dbIface)
	if err != nil {
		logger.Error("Failed to update interface", "interfaceID", iface.ID, "error", err)
		return err
	}
	return nil
}

func InterfacesChecks(net *db.Net, iface *Interface) error {
	if !net.VlanAware && iface.VlanTag != 0 {
		return errors.New("vlan_tag must be 0 for non-vlan-aware vnets")
	}

	reqIPAdd := ipaddr.NewIPAddressString(iface.IPAdd)

	if !reqIPAdd.IsPrefixed() {
		return errors.New("ip_add must have a subnet mask")
	}

	if iface.Gateway == "" {
		return nil
	}

	// Gateway checks

	if iface.Gateway == iface.IPAdd {
		return errors.New("gateway cannot be the same as ip_add")
	}

	zero, err := reqIPAdd.GetAddress().ToZeroHost()
	if err != nil {
		return errors.New("failed to get zero host address from ip_add")
	}

	reqGateway := ipaddr.NewIPAddressString(iface.Gateway).GetAddress()
	if !zero.PrefixContains(reqGateway) {
		return errors.New("gateway must be within ip subnet")
	}

	if reqGateway.IsPrefixed() {
		return errors.New("gateway must not have a subnet mask")
	}

	return nil
}
