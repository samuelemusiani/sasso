package db

import "time"

type Net struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	VNet   string `gorm:"not null;unique"` // Name of the VNet
	VNetID uint   `gorm:"not null;unique"` // ID of the VNet (VXLAN ID)

	Subnet    string `gorm:"not null;unique"` // Subnet of the VNet
	RouterIP  string `gorm:"not null;unique"` // Router IP of the VNet
	Broadcast string `gorm:"not null"`        // Broadcast address of the VNet
}

func initNets() error {
	err := db.AutoMigrate(&Net{})
	if err != nil {
		logger.With("error", err).Error("Failed to migrate Nets table")
		return err
	}
	return nil
}

func GetAllSubnets() ([]string, error) {
	var subnets []string
	result := db.Model(&Net{}).Pluck("subnet", &subnets)
	if result.Error != nil {
		return nil, result.Error
	}
	return subnets, nil
}
