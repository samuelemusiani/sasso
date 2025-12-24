package db

import "gorm.io/gorm"

type Subnet struct {
	ID     uint   `gorm:"primaryKey"`
	Subnet string `gorm:"not null"`

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

func NewSubnet(subnet string, peerID uint) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		var count int64

		err := tx.Model(&SubnetPeer{}).
			Joins("JOIN subnets ON subnets.id = subnet_peers.subnet_id").
			Joins("JOIN peers ON peers.id = subnet_peers.peer_id").
			Where("subnets.subnet = ? AND peers.id = ?", subnet, peerID).
			Count(&count).Error
		if err != nil {
			logger.Error("Failed to check existing subnet-peer association", "error", err)

			return err
		}

		if count > 0 {
			return ErrAlreadyExists
		}

		s := &Subnet{
			Subnet: subnet,
		}
		if err := tx.Create(s).Error; err != nil {
			logger.Error("Failed to create subnet", "error", err)

			return err
		}

		sp := &SubnetPeer{
			SubnetID: s.ID,
			PeerID:   peerID,
		}
		if err := tx.Create(sp).Error; err != nil {
			logger.Error("Failed to create subnet-peer association", "error", err)

			return err
		}

		return nil
	})

	return err
}

func RemoveSubnet(subnet string) error {
	return db.Where("subnet = ?", subnet).Delete(&Subnet{}).Error
}

func GetSubnetsByPeerID(peerID uint) ([]Subnet, error) {
	var subnets []Subnet
	if err := db.Joins("JOIN subnet_peers ON subnet_peers.subnet_id = subnets.id").
		Where("subnet_peers.peer_id = ?", peerID).
		Find(&subnets).Error; err != nil {
		return nil, err
	}

	return subnets, nil
}
