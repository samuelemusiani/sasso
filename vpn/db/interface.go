package db

import ()

type Interface struct {
	ID         uint   `gorm:"primaryKey"`
	PrivateKey string `gorm:"not null"`
	PublicKey  string `gorm:"not null"`
	Subnet     string `gorm:"not null;unique"`
	Address    string `gorm:"not null;unique"`
}

func initInterfaces() error {
	return db.AutoMigrate(&Interface{})
}

func NewInterface(privateKey, publicKey, subnet, address string) (*Interface, error) {
	iface := &Interface{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Subnet:     subnet,
		Address:    address,
	}
	if err := db.Create(iface).Error; err != nil {
		return nil, err
	}
	return iface, nil
}

func GetInterfaceByID(id uint) (*Interface, error) {
	var iface Interface
	if err := db.First(&iface, id).Error; err != nil {
		return nil, err
	}
	return &iface, nil
}

func GetAllAddresses() ([]string, error) {
	var addresses []string
	if err := db.Model(&Interface{}).Pluck("address", &addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}
