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
	Status   string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('running','stopped','suspended','unknown','deleting','creating','pre-deleting','pre-creating','configuring','pre-configuring')"`

	Name  string `gorm:"type:varchar(20);not null"`
	Notes string `gorm:"type:text;not null;default:''"`
	Cores uint   `gorm:"not null;default:1"`
	RAM   uint   `gorm:"not null;default:1024"`
	Disk  uint   `gorm:"not null;default:4"`

	IncludeGlobalSSHKeys bool `gorm:"not null"`

	Interfaces []Interface `gorm:"foreignKey:VMID;constraint:OnDelete:CASCADE"`
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

func ExistsVMWithUserIdAndName(userID uint, name string) (bool, error) {
	var count int64
	result := db.Model(&VM{}).Where("user_id = ? AND name = ?", userID, name).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func NewVM(ID uint64, userID uint, vmUserID uint, status string, name string, notes string, cores uint, ram uint, disk uint, includeGlobalSSHKeys bool) (*VM, error) {
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
	var t time.Time
	result := db.Where("status IN ?", states).Order("created_at DESC").Select("created_at").Limit(1).Scan(&t)
	if result.Error != nil {
		return time.Time{}, result.Error
	}
	return t, nil
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
