package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type VM struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Status string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('running','stopped','paused','unknown','deleting','creating','pre-deleting','pre-creating','configuring','pre-configuring')"`

	Name  string `gorm:"type:varchar(20);not null"`
	Notes string `gorm:"type:text;not null;default:''"`
	Cores uint   `gorm:"not null;default:1"`
	RAM   uint   `gorm:"not null;default:1024"`
	Disk  uint   `gorm:"not null;default:4"`

	LifeTime time.Time `gorm:"not null"`

	IncludeGlobalSSHKeys bool `gorm:"not null"`

	OwnerID   uint   `gorm:"not null;index"`
	OwnerType string `gorm:"not null;index"`

	Interfaces              []Interface                `gorm:"foreignKey:VMID;constraint:OnDelete:CASCADE"`
	ExpirationNotifications []VMExpirationNotification `gorm:"foreignKey:VMID;constraint:OnDelete:CASCADE"`
}

func initVMs() error {
	err := db.AutoMigrate(&VM{})
	if err != nil {
		logger.Error("Failed to migrate VMs table", "error", err)
		return err
	}

	return nil
}

func GetVMsByUserID(userID uint) ([]VM, error) {
	var vms []VM

	result := db.Where(&VM{OwnerID: userID, OwnerType: "User"}).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}

	return vms, nil
}

func GetVMsByGroupID(groupID uint) ([]VM, error) {
	var vms []VM

	result := db.Where(&VM{OwnerID: groupID, OwnerType: "Group"}).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}

	return vms, nil
}

func ExistsVMWithUserIDAndName(userID uint, name string) (bool, error) {
	return existsVMWithOwnerIDAndName(userID, "User", name)
}

func ExistsVMWithGroupIDAndName(groupID uint, name string) (bool, error) {
	return existsVMWithOwnerIDAndName(groupID, "Group", name)
}

// Returns true if a VM with the given userID and name exists
func existsVMWithOwnerIDAndName(ownerID uint, ownerType, name string) (bool, error) {
	var count int64

	result := db.Model(&VM{}).Where(&VM{OwnerID: ownerID, OwnerType: ownerType, Name: name}).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

func NewVMForUser(ID uint64, userID uint, status, name, notes string, cores, ram, disk uint, lifeTime time.Time, includeGlobalSSHKeys bool) (*VM, error) {
	return newvm(ID, userID, "User", status, name, notes, cores, ram, disk, lifeTime, includeGlobalSSHKeys)
}

func NewVMForGroup(ID uint64, groupID uint, status, name, notes string, cores, ram, disk uint, lifeTime time.Time, includeGlobalSSHKeys bool) (*VM, error) {
	return newvm(ID, groupID, "Group", status, name, notes, cores, ram, disk, lifeTime, includeGlobalSSHKeys)
}

func newvm(ID uint64, ownerID uint, ownerType string, status, name, notes string, cores, ram, disk uint, lifeTime time.Time, includeGlobalSSHKeys bool) (*VM, error) {
	vm := &VM{
		ID:                   ID,
		Status:               status,
		Name:                 name,
		Notes:                notes,
		Cores:                cores,
		RAM:                  ram,
		Disk:                 disk,
		LifeTime:             lifeTime,
		IncludeGlobalSSHKeys: includeGlobalSSHKeys,
		OwnerID:              ownerID,
		OwnerType:            ownerType,
	}

	result := db.Create(vm)
	if result.Error != nil {
		return nil, result.Error
	}

	return vm, nil
}

func GetVMByID(vmID uint64) (*VM, error) {
	var vm VM

	result := db.First(&vm, vmID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}

		return nil, result.Error
	}

	return &vm, nil
}

func DeleteVMByID(vmID uint64) error {
	result := db.Delete(&VM{}, vmID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func UpdateVMStatus(vmID uint64, status string) error {
	result := db.Model(&VM{ID: vmID}).Update("status", status)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return ErrNotFound
		}

		return result.Error
	}

	return nil
}

func UpdateVMResources(vmID uint64, cores, ram, disk uint) error {
	result := db.Model(&VM{ID: vmID}).
		UpdateColumns(VM{Cores: cores, RAM: ram, Disk: disk})

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return ErrNotFound
		}

		return result.Error
	}

	return nil
}

func GetVMByUserIDAndVMID(userID uint, vmID uint64) (*VM, error) {
	return getVMByOwnerIDAndVMID(userID, "User", vmID)
}

func GetVMByGroupIDAndVMID(groupID uint, vmID uint64) (*VM, error) {
	return getVMByOwnerIDAndVMID(groupID, "Group", vmID)
}

func getVMByOwnerIDAndVMID(ownerID uint, ownerType string, vmID uint64) (*VM, error) {
	var vm VM

	result := db.Where(&VM{OwnerID: ownerID, OwnerType: ownerType, ID: vmID}).First(&vm)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}

		return nil, result.Error
	}

	return &vm, nil
}

func GetVMsWithStatus(status string) ([]VM, error) {
	var vms []VM

	result := db.Where(&VM{Status: status}).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}

	return vms, nil
}

func GetVMsWithStates(states []string) ([]VM, error) {
	var vms []VM

	result := db.Where("status IN ?", states).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}

	return vms, nil
}

func GetTimeOfLastCreatedVMWithStates(states []string) (time.Time, error) {
	var vm VM

	result := db.Where("status IN ?", states).
		Order("created_at DESC").
		Limit(1).
		First(&vm)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return time.Time{}, nil // No VMs found with the specified states
		}

		return time.Time{}, result.Error
	}

	return vm.CreatedAt, nil
}

func GetAllActiveVMs() ([]VM, error) {
	return GetVMsWithStates([]string{"running", "stopped", "paused"})
}

func GetAllActiveVMsWithUnknown() ([]VM, error) {
	return GetVMsWithStates([]string{"running", "stopped", "paused", "unknown"})
}

func GetVMResourcesByUserID(userID uint) (uint, uint, uint, error) {
	return getVMResourcesByOwner(userID, "User")
}

func GetVMResourcesByGroupID(groupID uint) (uint, uint, uint, error) {
	return getVMResourcesByOwner(groupID, "Group")
}

func getVMResourcesByOwner(ownerID uint, ownerType string) (uint, uint, uint, error) {
	var result struct {
		Cores uint
		RAM   uint
		Disk  uint
	}

	err := db.Model(&VM{}).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk").
		Where(&VM{OwnerID: ownerID, OwnerType: ownerType}).Scan(&result).Error
	if err != nil {
		return 0, 0, 0, err
	}

	return result.Cores, result.RAM, result.Disk, nil
}

func GetResourcesActiveVMsByUserID(userID uint) (uint, uint, uint, error) {
	return getResourcesActiveVMsByOwner(userID, "User")
}

func GetResourcesActiveVMsByGroupID(groupID uint) (uint, uint, uint, error) {
	return getResourcesActiveVMsByOwner(groupID, "Group")
}

func getResourcesActiveVMsByOwner(ownerID uint, ownerType string) (uint, uint, uint, error) {
	var result struct {
		Cores uint
		RAM   uint
		Disk  uint
	}

	err := db.Model(&VM{}).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk").
		Where(&VM{OwnerID: ownerID, OwnerType: ownerType, Status: "running"}).Scan(&result).Error
	if err != nil {
		return 0, 0, 0, err
	}

	return result.Cores, result.RAM, result.Disk, nil
}

func CountVMs() (int64, error) {
	var count int64

	result := db.Model(&VM{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

func GetVMsWithLifetimesLessThan(t time.Time) ([]VM, error) {
	var vms []VM

	result := db.Where("life_time < ?", t).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}

	return vms, nil
}

func GetVMsWithLifetimesLessThanAndStatusIN(t time.Time, states []string) ([]VM, error) {
	var vms []VM

	result := db.Where("life_time < ? AND status IN ?", t, states).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}

	return vms, nil
}

func UpdateVMLifetime(vmID uint64, newLifetime time.Time) error {
	result := db.Model(&VM{ID: vmID}).Update("life_time", newLifetime)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return ErrNotFound
		}

		return result.Error
	}

	return nil
}

func GetAllVMsIDsByUserID(userID uint) ([]uint, error) {
	return getAllVMsIDsByOwner(userID, "User")
}

func GetAllVMsIDsByGroupID(goupID uint) ([]uint, error) {
	return getAllVMsIDsByOwner(goupID, "Group")
}

func getAllVMsIDsByOwner(ownerID uint, ownerType string) ([]uint, error) {
	var ids []uint

	err := db.Model(&VM{}).
		Where(&VM{OwnerID: ownerID, OwnerType: ownerType}).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

type VMExpirationNotification struct {
	ID         uint64 `gorm:"primaryKey"`
	VMID       uint64 `gorm:"not null;index"`
	DaysBefore uint   `gorm:"not null"`
}

func initVMExpirationNotifications() error {
	err := db.AutoMigrate(&VMExpirationNotification{})
	if err != nil {
		logger.Error("Failed to migrate VMExpirationNotifications table", "error", err)
		return err
	}

	return nil
}

func NewVMExpirationNotification(vmID uint64, daysBefore uint) (*VMExpirationNotification, error) {
	notification := &VMExpirationNotification{
		VMID:       vmID,
		DaysBefore: daysBefore,
	}

	result := db.Create(notification)
	if result.Error != nil {
		return nil, result.Error
	}

	return notification, nil
}

func GetVMExpirationNotificationsByVMID(vmID uint64) ([]VMExpirationNotification, error) {
	var notifications []VMExpirationNotification

	result := db.Model(&VMExpirationNotification{VMID: vmID}).Find(&notifications)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}

		return nil, result.Error
	}

	return notifications, nil
}

func CountGroupVMs(groupID uint) (int64, error) {
	var count int64

	result := db.Model(&VM{}).Where(&VM{OwnerID: groupID, OwnerType: "Group"}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}
