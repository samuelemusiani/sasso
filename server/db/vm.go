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
	DeletedAt *time.Time `gorm:"index"`

	UserID uint `gorm:"not null;uniqueIndex:idx_user_vm"`
	// VMUserID is an integer that counts the number of the VM for a specific user
	VMUserID uint   `gorm:"not null;default:0;uniqueIndex:idx_user_vm"`
	Status   string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('running','stopped','suspended','unknown','deleting','creating','pre-deleting','pre-creating','configuring','pre-configuring')"`

	Name  string `gorm:"type:varchar(20);not null"`
	Notes string `gorm:"type:text;not null;default:''"`
	Cores uint   `gorm:"not null;default:1"`
	RAM   uint   `gorm:"not null;default:1024"`
	Disk  uint   `gorm:"not null;default:4"`

	LifeTime time.Time `gorm:"not null"`

	IncludeGlobalSSHKeys bool `gorm:"not null"`

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
	result := db.Where("user_id = ?", userID).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}
	return vms, nil
}

// Returns true if a VM with the given userID and name exists
func ExistsVMWithUserIdAndName(userID uint, name string) (bool, error) {
	var count int64
	result := db.Model(&VM{}).Where("user_id = ? AND name = ?", userID, name).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func NewVM(ID uint64, userID uint, vmUserID uint, status string, name string, notes string, cores uint, ram uint, disk uint, lifeTime time.Time, includeGlobalSSHKeys bool) (*VM, error) {
	vm := &VM{
		ID:                   ID,
		UserID:               userID,
		VMUserID:             vmUserID,
		Status:               status,
		Name:                 name,
		Notes:                notes,
		Cores:                cores,
		RAM:                  ram,
		Disk:                 disk,
		LifeTime:             lifeTime,
		IncludeGlobalSSHKeys: includeGlobalSSHKeys,
	}
	result := db.Create(vm)
	if result.Error != nil {
		return nil, result.Error
	}
	return vm, nil
}

func GetLastVMUserIDByUserID(userID uint) (uint, error) {
	var vm VM
	result := db.Where("user_id = ?", userID).Order("vm_user_id DESC").First(&vm)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return 0, nil // No VMs found for this user
		}
		return 0, result.Error
	}
	return vm.VMUserID, nil
}

func GetVMByID(vmID uint64) (*VM, error) {
	var vm VM
	result := db.First(&vm, vmID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound // VM not found
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
		return gorm.ErrRecordNotFound
	}
	return nil
}

func UpdateVMStatus(vmID uint64, status string) error {
	result := db.Model(&VM{}).Where("id = ?", vmID).Update("status", status)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return result.Error
	}

	return nil
}

func GetVMByUserIDAndVMID(userID uint, vmID uint64) (*VM, error) {
	var vm VM
	result := db.Where("user_id = ? AND id = ?", userID, vmID).First(&vm)
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
	result := db.Where("status = ?", status).Find(&vms)
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
	var vms []VM
	statuses := []string{"running", "stopped", "suspended"}
	result := db.Where("status IN ?", statuses).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}
	return vms, nil
}

func GetAllActiveVMsWithUnknown() ([]VM, error) {
	var vms []VM
	statuses := []string{"running", "stopped", "suspended", "unknown"}
	result := db.Where("status IN ?", statuses).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}
	return vms, nil
}

func GetVMResourcesByUserID(userID uint) (uint, uint, uint, error) {
	var result struct {
		Cores uint
		RAM   uint
		Disk  uint
	}

	err := db.Model(&VM{}).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk").
		Where("user_id = ?", userID).Scan(&result).Error

	if err != nil {
		return 0, 0, 0, err
	}

	return result.Cores, result.RAM, result.Disk, nil
}

func GetResorcesActiveVMsByUserID(userID uint) (uint, uint, uint, error) {
	var result struct {
		Cores uint
		RAM   uint
		Disk  uint
	}

	err := db.Model(&VM{}).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk").
		Where("user_id = ? AND status = ?", userID, "running").Scan(&result).Error

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

func UpdateVMLifetime(vmID uint64, newLifetime time.Time) error {
	result := db.Model(&VM{}).Where("id = ?", vmID).Update("life_time", newLifetime)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return result.Error
	}

	return nil
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
	result := db.Where("vm_id = ?", vmID).Find(&notifications)
	if result.Error != nil {
		return nil, result.Error
	}
	return notifications, nil
}
