package db

import (
	"time"

	"gorm.io/gorm"
)

type Interface struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	LocalID   uint `gorm:"not null;"`
	CreatedAt time.Time
	UpdatedAt time.Time

	VNet   string `gorm:"not null;unique"` // Name of the VNet
	VNetID uint   `gorm:"not null;unique"` // ID of the VNet (VXLAN ID)

	Subnet    string `gorm:"not null;unique"` // Subnet of the VNet
	RouterIP  string `gorm:"not null;unique"` // Router IP of the VNet
	Broadcast string `gorm:"not null"`        // Broadcast address of the VNet

	FirewallInterfaceName string `gorm:"not null"` // Name of the interface on the firewall
}

func initInterfaces() error {
	return db.AutoMigrate(&Interface{})
}

func SaveInterface(iface Interface) error {
	return db.Create(&iface).Error
}

func GetAllUsedSubnets() ([]string, error) {
	var subnets []string
	if err := db.Model(&Interface{}).Pluck("subnet", &subnets).Error; err != nil {
		return nil, err
	}
	return subnets, nil
}

func GetInterfaceByVNet(vnet string) (*Interface, error) {
	var iface Interface
	if err := db.Where("v_net = ?", vnet).First(&iface).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.With("error", err).Error("Failed to retrieve interface by VNet")
		return nil, err
	}
	return &iface, nil
}

func GetInterfaceByVNetID(vnetID uint) (*Interface, error) {
	var iface Interface
	if err := db.Where("v_net_id = ?", vnetID).First(&iface).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.With("error", err).Error("Failed to retrieve interface by VNet ID")
		return nil, err
	}
	return &iface, nil
}

func DeleteInterface(id uint) error {
	return db.Delete(&Interface{}, id).Error
}

func UpdateInterface(iface Interface) error {
	return db.Save(&iface).Error
}
