package db

import "gorm.io/gorm"

type Net struct {
	ID   uint   `gorm:"primaryKey"`
	Zone string `gorm:"not null"`
	Name string `gorm:"not null"`
	Tag  uint32 `gorm:"not null"`

	Subnet    string `gorm:"not null"` // CIDR notation of the subnet
	Gateway   string `gorm:"not null"` // IP address of the gateway
	Broadcast string `gorm:"not null"` // Broadcast address of the subnet

	UserIDs []NetUserID `gorm:"foreignKey:NetID;constraint:OnDelete:CASCADE;"`
}

type NetUserID struct {
	NetID  uint `gorm:"primaryKey"`
	UserID uint `gorm:"primaryKey"`
}

func initNets() error {
	return db.AutoMigrate(&Net{}, &NetUserID{})
}

func UpdateAllNets(nets []Net) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM net_user_ids").Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM nets").Error; err != nil {
			return err
		}

		for _, net := range nets {
			if err := tx.Create(&net).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func GetAllNets() ([]Net, error) {
	var nets []Net
	if err := db.Preload("UserIDs").Find(&nets).Error; err != nil {
		return nil, err
	}

	return nets, nil
}
