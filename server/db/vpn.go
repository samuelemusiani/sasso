package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type VPNConfig struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	IP              string         `gorm:"not null;uniqueIndex"`
	PeerPrivateKey  string         `gorm:"not null"`
	ServerPublicKey string         `gorm:"not null"`
	Endpoint        string         `gorm:"not null"`
	AllowedIPs      []VPNAllowedIP `gorm:"foreignKey:VPNConfigID;constraint:OnDelete:CASCADE"`

	UserID uint `gorm:"not null;index"`
}

type VPNAllowedIP struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	VPNConfigID uint   `gorm:"not null;index"`
	IP          string `gorm:"not null"`
}

func initVPNConfig() error {
	return db.AutoMigrate(&VPNConfig{})
}

func GetVPNConfigByID(id uint) (*VPNConfig, error) {
	var vpnConfig VPNConfig

	result := db.Preload("AllowedIPs").First(&vpnConfig, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		logger.Error("Failed to retrieve VPN config by ID", "error", result.Error)

		return nil, result.Error
	}

	return &vpnConfig, nil
}

func GetVPNConfigsByUserID(userID uint) ([]VPNConfig, error) {
	var vpnConfigs []VPNConfig

	result := db.Where(&VPNConfig{UserID: userID}).Preload("VPNConfig").Find(&vpnConfigs)
	if result.Error != nil {
		logger.Error("Failed to retrieve VPN configs by user ID", "error", result.Error)

		return nil, result.Error
	}

	return vpnConfigs, nil
}

func GetVPNConfigByIP(ip string) (*VPNConfig, error) {
	var vpnConfig VPNConfig

	result := db.Preload("VPNConfig").First(&vpnConfig, &VPNConfig{IP: ip})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		logger.Error("Failed to retrieve VPN config by IP", "error", result.Error)

		return nil, result.Error
	}

	return &vpnConfig, nil
}

func CreateVPNConfig(config VPNConfig) error {
	result := db.Create(config)
	if result.Error != nil {
		logger.Error("Failed to create VPN config", "error", result.Error)

		return result.Error
	}

	return nil
}

func UpdateVPNConfig(config VPNConfig) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Updates(&config).Error; err != nil {
			logger.Error("Failed to update VPN config by ID", "error", err)

			return err
		}

		if err := tx.Model(&config).Association("AllowedIPs").Replace(config.AllowedIPs); err != nil {
			logger.Error("Failed to update VPN allowed IPs", "error", err)

			return err
		}

		return nil
	})
}

func GetAllVPNConfigs() ([]VPNConfig, error) {
	var vpnConfigs []VPNConfig

	result := db.Preload("AllowedIPs").Find(&vpnConfigs)
	if result.Error != nil {
		logger.Error("Failed to retrieve all VPN configs", "error", result.Error)

		return nil, result.Error
	}

	return vpnConfigs, nil
}

func CountVPNConfigsByUserID(userID uint) (int64, error) {
	var count int64

	result := db.Model(&VPNConfig{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		logger.Error("Failed to count VPN configs by user ID", "error", result.Error)

		return 0, result.Error
	}

	return count, nil
}

func DeleteVPNConfigByID(id uint) error {
	result := db.Delete(&VPNConfig{}, id)
	if result.Error != nil {
		logger.Error("Failed to delete VPN config by ID", "error", result.Error)

		return result.Error
	}

	return nil
}
