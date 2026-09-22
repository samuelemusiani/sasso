package db

import (
	"time"

	"gorm.io/gorm"
)

type WireguardPeer struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	IP              string `gorm:"not null;unique"`
	PeerPrivateKey  string `gorm:"not null"`
	ServerPublicKey string `gorm:"not null"`
	Endpoint        string `gorm:"not null"`

	AllowedIPs []WireguardAllowedIP `gorm:"foreignKey:WireguardPeerID;constraint:OnDelete:CASCADE;"`

	UserID uint `gorm:"index"`
}

type WireguardAllowedIP struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	WireguardPeerID uint   `gorm:"not null;index"`
	IP              string `gorm:"not null"`
}

func initWireguardPeers() error {
	return db.AutoMigrate(&WireguardPeer{}, &WireguardAllowedIP{})
}

func GetAllWireguardPeers() ([]WireguardPeer, error) {
	var peers []WireguardPeer

	if err := db.Preload("AllowedIPs").Find(&peers).Error; err != nil {
		return nil, err
	}

	return peers, nil
}

func UpdateAllWireguardPeers(configs []WireguardPeer) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM wireguard_peers").Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM wireguard_allowed_ips").Error; err != nil {
			return err
		}

		for _, config := range configs {
			if err := tx.Create(&config).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func GetAllAddresses() ([]string, error) {
	var addresses []string

	if err := db.Model(&WireguardPeer{}).Pluck("ip", &addresses).Error; err != nil {
		return nil, err
	}

	return addresses, nil
}
