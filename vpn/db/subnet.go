package db

import "gorm.io/gorm"

type Subnet struct {
	ID     uint   `gorm:"primaryKey"`
	Subnet string `gorm:"not null;unique"`

	Peers []Peer `gorm:"many2many:subnet_peers;constraint:OnDelete:CASCADE;"`
}

type SubnetPeer struct {
	SubnetID uint `gorm:"primaryKey"`
	PeerID   uint `gorm:"primaryKey"`
}

func initSubnets() error {
	return db.AutoMigrate(&Subnet{})
}

func GetAllSubnets() ([]Subnet, error) {
	var subnets []Subnet
	if err := db.Preload("Peers").Find(&subnets).Error; err != nil {
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
	err := db.Transaction(func(tx *gorm.DB) error {
		s := &Subnet{
			Subnet: subnet,
		}
		if err := db.Create(s).Error; err != nil {
			return err
		}

		sp := &SubnetPeer{
			SubnetID: s.ID,
			PeerID:   PeerID,
		}
		if err := db.Create(sp).Error; err != nil {
			return err
		}

		return nil
	})
	return err
}

func RemoveSubnet(subnet string) error {
	if err := db.Where("subnet = ?", subnet).Delete(&Subnet{}).Error; err != nil {
		return err
	}
	return nil
}
