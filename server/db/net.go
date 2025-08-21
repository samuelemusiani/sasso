package db

import (
	"gorm.io/gorm"
)

type Net struct {
	gorm.Model
	Name      string `gorm:"uniqueIndex;not null"`
	Alias     string `gorm:"not null"`             // For users
	Zone      string `gorm:"not null"`             // Should be "sasso"
	Tag       uint32 `gorm:"not null;uniqueIndex"` // Unique tag for the network
	VlanAware bool   `gorm:"not null;default:false"`

	UserID uint   `gorm:"not null;uniqueIndex:idx_user_net"`
	Status string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('unknown','creating','deleting','pre-creating','pre-deleting')"`
}

func initNetworks() error {
	if err := db.AutoMigrate(&Net{}); err != nil {
		logger.With("error", err).Error("Failed to migrate networks table")
		return err
	}
	logger.Info("Networks table migrated successfully")
	return nil
}

func GetNetByID(ID uint) (*Net, error) {
	var net Net
	if err := db.First(&net, ID).Error; err != nil {
		logger.With("netID", ID, "error", err).Error("Failed to find network by ID")
		return nil, err
	}
	return &net, nil
}

func GetNetByName(name string) (*Net, error) {
	var net Net
	if err := db.Where("name = ?", name).First(&net).Error; err != nil {
		logger.With("netName", name, "error", err).Error("Failed to find network by name")
		return nil, err
	}
	return &net, nil
}

func GetLastUsedTagByZone(zone string) (uint32, error) {
	var net Net
	if err := db.Where("zone = ?", zone).Order("tag DESC").First(&net).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil // No networks found for this zone
		}
		logger.With("zone", zone, "error", err).Error("Failed to get last used tag by zone")
		return 0, err
	}
	return net.Tag, nil
}

func GetNetsByUserID(userID uint) ([]Net, error) {
	var nets []Net
	if err := db.Where("user_id = ?", userID).Find(&nets).Error; err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get nets for user")
		return nil, err
	}
	return nets, nil
}

func CreateNetForUser(userID uint, name, alias, zone string, tag uint32, vlanAware bool, status string) (*Net, error) {
	// This function only creates a network for a user in the DB. It does
	// not create the network in Proxmox

	net := &Net{
		Name:      string(name[:]),
		Alias:     alias,
		Zone:      zone,
		Tag:       tag,
		VlanAware: vlanAware,
		UserID:    userID,
		Status:    status,
	}

	if err := db.Create(net).Error; err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to create network for user")
		return nil, err
	}

	logger.With("userID", userID, "netName", net.Name, "zone", net.Zone, "tag", net.Tag, "vlanAware", net.VlanAware).Info("Created network for user")

	return net, nil
}

func GetVNetsWithStatus(status string) ([]Net, error) {
	var nets []Net
	if err := db.Where("status = ?", status).Find(&nets).Error; err != nil {
		logger.With("status", status, "error", err).Error("Failed to get VNets with status")
		return nil, err
	}
	return nets, nil
}

func UpdateVNetStatus(ID uint, status string) error {
	if err := db.Model(&Net{}).Where("id = ?", ID).Update("status", status).Error; err != nil {
		logger.With("netID", ID, "status", status, "error", err).Error("Failed to update VNet status")
		return err
	}
	logger.With("netID", ID, "status", status).Info("Updated VNet status")
	return nil
}

func DeleteNetByID(ID uint) error {
	if err := db.Delete(&Net{}, ID).Error; err != nil {
		logger.With("netID", ID, "error", err).Error("Failed to delete network")
		return err
	}
	logger.With("netID", ID).Info("Deleted network")
	return nil
}
