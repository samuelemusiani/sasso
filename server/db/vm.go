package db

import "gorm.io/gorm"

type VM struct {
	gorm.Model
	UserID uint   `gorm:"not null"`
	Status string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('running','stopped','suspended','unknown')"`
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
