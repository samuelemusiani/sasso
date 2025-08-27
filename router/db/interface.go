package db

import "time"

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
