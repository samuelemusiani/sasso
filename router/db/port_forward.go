package db

import (
	"errors"

	"gorm.io/gorm"
)

type PortForward struct {
	ID       uint   `gorm:"primaryKey"`
	OutPort  uint16 `gorm:"not null; uniqueIndex"`
	DestPort uint16 `gorm:"not null"`
	DestIP   string `gorm:"not null"`
}

func initPortForwards() error {
	return db.AutoMigrate(&PortForward{})
}

func GetPortForwards() ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Find(&pfs).Error; err != nil {
		return nil, err
	}

	return pfs, nil
}

func GetPortForwardByID(pfID uint) (*PortForward, error) {
	var pf PortForward
	if err := db.First(&pf, pfID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &pf, nil
}

func AddPortForward(pf PortForward) error {
	return db.Create(&pf).Error
}

func RemovePortForward(pfID uint) error {
	return db.Delete(&PortForward{}, pfID).Error
}
