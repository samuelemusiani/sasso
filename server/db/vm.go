package db

import (
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
	Status   string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('running','stopped','suspended','unknown')"`
}

func initVMs() error {
	err := db.AutoMigrate(&VM{})
	if err != nil {
		logger.With("error", err).Error("Failed to migrate VMs table")
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

func NewVM(ID uint64, userID uint, vmUserID uint, status string) (*VM, error) {
	vm := &VM{
		ID:       ID,
		UserID:   userID,
		VMUserID: vmUserID,
		Status:   status,
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
