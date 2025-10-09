package proxmox

import "samuelemusiani/sasso/server/db"

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

func NewInterface(VMID uint, vnetID uint, vlanTag uint16, ipAdd string, gateway string) (*Interface, error) {
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
	err := db.UpdateInterfaceStatus(id, string(InterfaceStatusPreDeleting))
	if err != nil {
		logger.Error("Failed to set interface status to pre-deleting", "interfaceID", id, "error", err)
		return err
	}
	return nil
}
