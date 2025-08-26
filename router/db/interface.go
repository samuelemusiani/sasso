package db

type Interface struct {
	ID      uint `gorm:"primaryKey;autoIncrement"`
	LocalID uint `gorm:"not null;"`
	VNet    string
	VNetID  uint
}

func initInterfaces() error {
	return db.AutoMigrate(&Interface{})
}

func SaveInterface(iface Interface) error {
	return db.Create(&iface).Error
}
