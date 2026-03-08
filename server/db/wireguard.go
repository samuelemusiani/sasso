package db

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type WireguardPeer struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	IP              string `gorm:"not null;unique"`
	PeerPrivateKey  string `gorm:"not null"`
	ServerPublicKey string `gorm:"not null"`
	Endpoint        string `gorm:"not null"`

	AllowedIPs []WireguardAllowedIP `gorm:"foreignKey:WireguardPeerID;constraint:OnDelete:CASCADE;"`

	UserID uint `gorm:"not null;index"`
}

type WireguardAllowedIP struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	WireguardPeerID uint   `gorm:"not null;index"`
	IP              string `gorm:"type:varchar(45);not null"`
}

func initWireguardPeers() error {
	return db.AutoMigrate(&WireguardPeer{}, &WireguardAllowedIP{})
}

func GetWireguardPeerByID(id uint) (*WireguardPeer, error) {
	var vpnConfig WireguardPeer

	result := db.Preload("AllowedIPs").First(&vpnConfig, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to retrieve wireguard peer by ID: %w", result.Error)
	}

	return &vpnConfig, nil
}

func GetWireguardPeerByUserID(userID uint) ([]WireguardPeer, error) {
	var vpnConfigs []WireguardPeer

	result := db.Where("user_id = ?", userID).Preload("AllowedIPs").Find(&vpnConfigs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve wireguard peers by user ID: %w", result.Error)
	}

	return vpnConfigs, nil
}

func GetWireguardPeerByIP(ip string) (*WireguardPeer, error) {
	var vpnConfig WireguardPeer

	result := db.Where("ip = ?", ip).Preload("AllowedIPs").First(&vpnConfig)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to retrieve wireguard peer by IP: %w", result.Error)
	}

	return &vpnConfig, nil
}

func CreateWireguardPeer(userID uint) error {
	result := db.Create(&WireguardPeer{UserID: userID})
	if result.Error != nil {
		return fmt.Errorf("failed to create wireguard peer: %w", result.Error)
	}

	return nil
}

func UpdateWireguardPeer(wgPeer *WireguardPeer) error {
	result := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(wgPeer)
	if result.Error != nil {
		return fmt.Errorf("failed to update wireguard peer: %w", result.Error)
	}

	return nil
}

func GetAllWireguardPeers() ([]WireguardPeer, error) {
	var vpnConfigs []WireguardPeer

	result := db.Preload("AllowedIPs").Find(&vpnConfigs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve all wireguard peers: %w", result.Error)
	}

	return vpnConfigs, nil
}

func CountWireguardPeersByUserID(userID uint) (int64, error) {
	var count int64

	result := db.Model(&WireguardPeer{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count wireguard peers by user ID: %w", result.Error)
	}

	return count, nil
}

func DeleteWireguardPeerByID(id uint) error {
	result := db.Delete(&WireguardPeer{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete wireguard peer by ID: %w", result.Error)
	}

	return nil
}

func DeleteWireguardPeerByIDAndUserID(id uint, userID uint) error {
	result := db.Where("id = ? AND user_id = ?", id, userID).Delete(&WireguardPeer{})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}

		return fmt.Errorf("failed to delete wireguard peer by ID and user ID: %w", result.Error)
	}

	return nil
}
