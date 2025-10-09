package db

type Subnet struct {
	ID     uint   `gorm:"primaryKey"`
	Subnet string `gorm:"not null;unique"`

	PeerID uint `gorm:"index; not null"`
}

func initSubnets() error {
	return db.AutoMigrate(&Subnet{})
}

func GetAllSubnets() ([]Subnet, error) {
	var subnets []Subnet
	if err := db.Find(&subnets).Error; err != nil {
		return nil, err
	}
	return subnets, nil
}

func CheckSubnetExists(subnet string) (bool, error) {
	var count int64
	if err := db.Model(&Subnet{}).Where("subnet = ?", subnet).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func NewSubnet(subnet string, PeerID uint) error {
	s := &Subnet{
		Subnet: subnet,
		PeerID: PeerID,
	}
	if err := db.Create(s).Error; err != nil {
		return err
	}
	return nil
}

func RemoveSubnet(subnet string) error {
	if err := db.Where("subnet = ?", subnet).Delete(&Subnet{}).Error; err != nil {
		return err
	}
	return nil
}
