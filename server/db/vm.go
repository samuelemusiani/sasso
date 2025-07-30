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
	Status   string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('running','stopped','suspended','unknown','deleting','creating','pre-deleting','pre-creating')"`
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

func GetAllActiveVMs() ([]VM, error) {
	var vms []VM
	statuses := []string{"running", "stopped", "suspended"}
	result := db.Where("status IN ?", statuses).Find(&vms)
	if result.Error != nil {
		return nil, result.Error
	}
	return vms, nil
}
