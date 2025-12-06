package db

import (
	"time"

	"gorm.io/gorm"
)

type VPNConfig struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	VPNConfig string `gorm:"type:text;not null"`
	VPNIP     string `gorm:"type:varchar(45);not null"`

	UserID uint `gorm:"not null;index"`
}

func initVPNConfig() error {
	return db.AutoMigrate(&VPNConfig{})
}

func GetVPNConfigByID(id uint) (*VPNConfig, error) {
	var vpnConfig VPNConfig
	result := db.First(&vpnConfig, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		} else {
			logger.Error("Failed to retrieve VPN config by ID", "error", result.Error)
			return nil, result.Error
		}
	}

	return &vpnConfig, nil
}

func GetVPNConfigsByUserID(userID uint) ([]VPNConfig, error) {
	var vpnConfigs []VPNConfig
	result := db.Where("user_id = ?", userID).Find(&vpnConfigs)
	if result.Error != nil {
		logger.Error("Failed to retrieve VPN configs by user ID", "error", result.Error)
		return nil, result.Error
	}

	return vpnConfigs, nil
}

func GetVPNConfigByIP(vpnIP string) (*VPNConfig, error) {
	var vpnConfig VPNConfig
	result := db.First(&vpnConfig, "vpn_ip = ?", vpnIP)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		} else {
			logger.Error("Failed to retrieve VPN config by IP", "error", result.Error)
			return nil, result.Error
		}
	}

	return &vpnConfig, nil
}

func CreateVPNConfig(vpnConfig string, vpnIP string, userID uint) error {
	vpn := &VPNConfig{
		VPNConfig: vpnConfig,
		VPNIP:     vpnIP,
		UserID:    userID,
	}
	result := db.Create(vpn)
	if result.Error != nil {
		logger.Error("Failed to create VPN config", "error", result.Error)
		return result.Error
	}
	return nil
}

func UpdateVPNConfigByID(id uint, newConfig string, newIP string) error {
	result := db.Model(&VPNConfig{}).Where("id = ?", id).Updates(VPNConfig{VPNConfig: newConfig, VPNIP: newIP})
	if result.Error != nil {
		logger.Error("Failed to update VPN config by ID", "error", result.Error)
		return result.Error
	}
	return nil
}

func GetAllVPNConfigs() ([]VPNConfig, error) {
	var vpnConfigs []VPNConfig
	result := db.Find(&vpnConfigs)
	if result.Error != nil {
		logger.Error("Failed to retrieve all VPN configs", "error", result.Error)
		return nil, result.Error
	}

	return vpnConfigs, nil
}

func DeleteVPNConfigByID(id uint) error {
	result := db.Delete(&VPNConfig{}, id)
	if result.Error != nil {
		logger.Error("Failed to delete VPN config by ID", "error", result.Error)
		return result.Error
	}
	return nil
}
