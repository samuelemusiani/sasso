package db

import (
	"fmt"
	"strings"
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

	// read-only, not stored in DB
	VNetName  string `gorm:"->;-:migration"` // Name of the VNet
	VMName    string `gorm:"->;-:migration"` // Name of the VM
	GroupID   uint   `gorm:"->;-:migration"` // Group ID of the owner of the VM
	GroupName string `gorm:"->;-:migration"` // Group Name of the owner of the VM
	GroupRole string `gorm:"->;-:migration"` // Role of the user in the group
}

func initInterfaces() error {
	if err := db.AutoMigrate(&Interface{}); err != nil {
		return fmt.Errorf("failed to migrate interfaces table: %w", err)
	}

	logger.Debug("Interfaces table migrated successfully")

	return nil
}

func GetInterfaceByID(id uint) (*Interface, error) {
	var iface Interface
	if err := db.First(&iface, id).Error; err != nil {
		return nil, fmt.Errorf("failed to find interface by ID: %w", err)
	}

	return &iface, nil
}

func GetInterfacesByVMID(vmID uint64) ([]Interface, error) {
	var ifaces []Interface
	if err := db.Where("vm_id = ?", vmID).Find(&ifaces).Error; err != nil {
		return nil, fmt.Errorf("failed to get interfaces for VM: %w", err)
	}

	return ifaces, nil
}

func GetInterfacesWithStatus(status string) ([]Interface, error) {
	var ifaces []Interface
	if err := db.Where("status = ?", status).Find(&ifaces).Error; err != nil {
		return nil, fmt.Errorf("failed to get interfaces with status: %w", err)
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
		return nil, fmt.Errorf("failed to create interface: %w", result.Error)
	}

	return iface, nil
}

func UpdateInterface(iface *Interface) error {
	if err := db.Save(iface).Error; err != nil {
		return fmt.Errorf("failed to update interface: %w", err)
	}

	return nil
}

func UpdateInterfaceStatus(id uint, status string) error {
	if err := db.Model(&Interface{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update interface status: %w", err)
	}

	return nil
}

func DeleteInterfaceByID(id uint) error {
	if err := db.Delete(&Interface{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete interface by ID: %w", err)
	}

	return nil
}

func DeleteInterface(iface *Interface) error {
	if err := db.Delete(iface).Error; err != nil {
		return fmt.Errorf("failed to delete interface: %w", err)
	}

	return nil
}

func GetInterfacesByVNetID(vnetID uint) ([]Interface, error) {
	var ifaces []Interface
	if err := db.Where("v_net_id = ?", vnetID).Find(&ifaces).Error; err != nil {
		return nil, fmt.Errorf("failed to get interfaces for VNet: %w", err)
	}

	return ifaces, nil
}

func DeleteAllInterfacesByVMID(vmID uint64) error {
	if err := db.Where("vm_id = ?", vmID).Delete(&Interface{}).Error; err != nil {
		return fmt.Errorf("failed to delete all interfaces by VM ID: %w", err)
	}

	return nil
}

func AreThereInterfacesWithVlanTagsByVNetID(vnetID uint) (bool, error) {
	var count int64
	if err := db.Model(&Interface{}).Where("v_net_id = ? AND vlan_tag != 0", vnetID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to count interfaces with VLAN tag for VNet: %w", err)
	}

	return count > 0, nil
}

func CountInterfacesWithStates() ([]StatusCount, error) {
	var counts []StatusCount

	result := db.Model(&Interface{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&counts)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to count interfaces with states: %w", result.Error)
	}

	return counts, nil
}

func CountInterfacesOnVM(vmID uint) (int64, error) {
	var count int64
	if err := db.Model(&Interface{}).Where("vm_id = ?", vmID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count interfaces on VM: %w", err)
	}

	return count, nil
}

func GetAllInterfacesWithExtrasByUserID(userID uint) ([]Interface, error) {
	var ifaces []Interface

	//nolint:unqueryvet // schema is generated from struct via GORM, columns are always in sync
	query := db.Raw(`SELECT interfaces.*, vms.name as vm_name, nets.alias as v_net_name, user_groups.role as group_role, groups.name as group_name, groups.id as group_id
		FROM interfaces
		JOIN vms ON vms.id = interfaces.vm_id
		JOIN nets ON nets.id = interfaces.v_net_id
		LEFT JOIN user_groups on vms.owner_id = user_groups.group_id AND vms.owner_type = 'Group'
		LEFT JOIN groups on user_groups.group_id = groups.id
		WHERE (vms.owner_id = ? AND vms.owner_type = 'User')
			OR (vms.owner_type = 'Group' AND user_groups.user_id = ?)`, userID, userID)
	if err := query.Scan(&ifaces).Error; err != nil {
		return nil, fmt.Errorf("failed to get interfaces with extras by user ID: %w", err)
	}

	return ifaces, nil
}

func ExistsIPInVNetWithVlanTag(vnetID uint, vlanTag uint16, ipAdd string) (bool, error) {
	if slashIndex := strings.Index(ipAdd, "/"); slashIndex != -1 {
		ipAdd = ipAdd[:slashIndex]
	}

	ipAdd += "/%"

	var count int64
	if err := db.Model(&Interface{}).
		Where("v_net_id = ? AND vlan_tag = ? AND ip_add LIKE ?", vnetID, vlanTag, ipAdd).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check existence of IP in VNet with VLAN tag: %w", err)
	}

	return count > 0, nil
}
