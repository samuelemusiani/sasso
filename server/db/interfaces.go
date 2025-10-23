package db

import (
	"time"
)

type Interface struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	LocalID uint   `gorm:"not null"` // Local unique ID for the interface
	VMID    uint   `gorm:"not null"`
	VNetID  uint   `gorm:"not null"`
	VlanTag uint16 `gorm:"not null;default:0"` // 0 means untagged
	IPAdd   string `gorm:"not null"`
	Gateway string `gorm:"not null"`

	Status string `gorm:"type:varchar(20);not null;default:'creating';check:status IN ('unknown','pre-creating','creating','pre-deleting','deleting','ready','pre-configuring','configuring')"`
}

func initInterfaces() error {
	if err := db.AutoMigrate(&Interface{}); err != nil {
		logger.Error("Failed to migrate interfaces table", "error", err)
		return err
	}
	logger.Debug("Interfaces table migrated successfully")
	return nil
}

func GetInterfaceByID(ID uint) (*Interface, error) {
	var iface Interface
	if err := db.First(&iface, ID).Error; err != nil {
		logger.Error("Failed to find interface by ID", "ifaceID", ID, "error", err)
		return nil, err
	}
	return &iface, nil
}

func GetInterfacesByVMID(vmID uint64) ([]Interface, error) {
	var ifaces []Interface
	if err := db.Where("vm_id = ?", vmID).Find(&ifaces).Error; err != nil {
		logger.Error("Failed to get interfaces for VM", "vmID", vmID, "error", err)
		return nil, err
	}
	return ifaces, nil
}

func GetInterfacesWithStatus(status string) ([]Interface, error) {
	var ifaces []Interface
	if err := db.Where("status = ?", status).Find(&ifaces).Error; err != nil {
		logger.Error("Failed to get interfaces with status", "status", status, "error", err)
		return nil, err
	}
	return ifaces, nil
}

func NewInterface(vmID uint, vNetID uint, vlanTag uint16, ipAdd string, gateway string, status string) (*Interface, error) {
	iface := &Interface{
		VMID:    vmID,
		VNetID:  vNetID,
		VlanTag: vlanTag,
		IPAdd:   ipAdd,
		Gateway: gateway,
		Status:  status,
	}
	result := db.Create(iface)
	if result.Error != nil {
		return nil, result.Error
	}
	return iface, nil
}

func UpdateInterface(iface *Interface) error {
	return db.Save(iface).Error
}

func UpdateInterfaceStatus(id uint, status string) error {
	return db.Model(&Interface{}).Where("id = ?", id).Update("status", status).Error
}

func DeleteInterfaceByID(id uint) error {
	return db.Delete(&Interface{}, id).Error
}

func DeleteInterface(iface *Interface) error {
	return db.Delete(iface).Error
}

func GetInterfacesByVNetID(vnetID uint) ([]Interface, error) {
	var ifaces []Interface
	if err := db.Where("v_net_id = ?", vnetID).Find(&ifaces).Error; err != nil {
		logger.Error("Failed to get interfaces for VNet", "vnetID", vnetID, "error", err)
		return nil, err
	}
	return ifaces, nil
}

func DeleteAllInterfacesByVMID(vmID uint64) error {
	return db.Where("vm_id = ?", vmID).Delete(&Interface{}).Error
}

func AreThereInterfacesWithVlanTagsByVNetID(vnetID uint) (bool, error) {
	var count int64
	if err := db.Model(&Interface{}).Where("v_net_id = ? AND vlan_tag != 0", vnetID).Count(&count).Error; err != nil {
		logger.Error("Failed to count interfaces with VLAN tag for VNet", "vnetID", vnetID, "error", err)
		return false, err
	}
	return count > 0, nil
}

func CountInterfaces() (int64, error) {
	var count int64
	if err := db.Model(&Interface{}).Count(&count).Error; err != nil {
		logger.Error("Failed to count interfaces", "error", err)
		return 0, err
	}
	return count, nil
}
