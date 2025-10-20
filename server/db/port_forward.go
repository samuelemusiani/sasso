package db

import (
	"time"
)

type PortForward struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	OutPort  uint16 `gorm:"not null; uniqueIndex"`
	DestPort uint16 `gorm:"not null"`
	DestIP   string `gorm:"not null"`
	Approved bool   `gorm:"not null;default:false"`

	OwnerID   uint   `gorm:"not null;index"`
	OwnerType string `gorm:"not null;index"`

	VNetID uint `gorm:"not null"`

	// Used only during joins for group name or username
	Name  string `gorm:"->;-:migration"`
	Group bool   `gorm:"->;-:migration"`
}

func initPortForwards() error {
	if err := db.AutoMigrate(&PortForward{}); err != nil {
		logger.Error("Failed to migrate port forwards table", "error", err)
		return err
	}
	logger.Debug("Port forwards table migrated successfully")
	return nil
}

func GetPortForwards() ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Find(&pfs).Error; err != nil {
		logger.Error("Failed to get all port forwards", "error", err)
		return nil, err
	}
	return pfs, nil
}

func GetPortForwardsWithNames() ([]PortForward, error) {
	var portForwards []PortForward
	err := db.Table("port_forwards pf").
		Select(`pf.*, 
           COALESCE(users.username, groups.name) as name,
           CASE WHEN pf.owner_type = ? THEN true ELSE false END as group`, "Group").
		Joins("LEFT JOIN users ON pf.owner_type = ? AND pf.owner_id = users.id", "User").
		Joins("LEFT JOIN groups ON pf.owner_type = ? AND pf.owner_id = groups.id", "Group").
		Find(&portForwards).Error
	if err != nil {
		logger.Error("Failed to get port forwards with usernames", "error", err)
		return nil, err
	}
	return portForwards, nil
}

func GetApprovedPortForwards() ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Where("approved = ?", true).Find(&pfs).Error; err != nil {
		logger.Error("Failed to get approved port forwards", "error", err)
		return nil, err
	}
	return pfs, nil
}

func GetPortForwardByID(ID uint) (*PortForward, error) {
	var pf PortForward
	if err := db.First(&pf, ID).Error; err != nil {
		logger.Error("Failed to find port forward by ID", "pfID", ID, "error", err)
		return nil, err
	}
	return &pf, nil
}

func GetPortForwardsByUserID(userID uint) ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Where(&PortForward{OwnerID: userID}).Find(&pfs).Error; err != nil {
		logger.Error("Failed to get port forwards for user", "userID", userID, "error", err)
		return nil, err
	}
	return pfs, nil
}

func AddPortForwardForUser(outPort, destPort uint16, destIP, subnet string, userID uint) (*PortForward, error) {
	return addPortForwardForOwner(outPort, destPort, destIP, subnet, userID, "User")
}

func AddPortForwardForGroup(outPort, destPort uint16, destIP, subnet string, groupID uint) (*PortForward, error) {
	return addPortForwardForOwner(outPort, destPort, destIP, subnet, groupID, "Group")
}

func addPortForwardForOwner(outPort, destPort uint16, destIP, subnet string, ownerID uint, ownerType string) (*PortForward, error) {

	net, err := GetVNetBySubnet(subnet)
	if err != nil {
		logger.Error("Failed to find VNet by subnet", "subnet", subnet, "error", err)
		return nil, err
	}

	pf := &PortForward{
		OutPort:   outPort,
		DestPort:  destPort,
		DestIP:    destIP,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		Approved:  false,
		VNetID:    net.ID,
	}
	if err := db.Create(pf).Error; err != nil {
		logger.Error("Failed to create port forward", "error", err)
		return nil, err
	}
	return pf, nil
}

func UpdatePortForwardApproval(pfID uint, approve bool) error {
	if err := db.Model(&PortForward{}).Where("id = ?", pfID).Update("approved", approve).Error; err != nil {
		logger.Error("Failed to update port forward approval", "pfID", pfID, "error", err)
		return err
	}
	return nil
}

func GetRandomAvailableOutPort(start, end uint16) (uint16, error) {
	var outPort int
	query := `
		SELECT p FROM generate_series(?::integer, ?::integer) AS p
		LEFT JOIN port_forwards pf ON pf.out_port = p
		WHERE pf.out_port IS NULL
		ORDER BY RANDOM()
		LIMIT 1;
	`
	err := db.Raw(query, start, end).Scan(&outPort).Error
	if err != nil {
		logger.Error("Failed to get random available out port", "error", err)
		return 0, err
	}
	return uint16(outPort), nil
}

func DeletePortForward(pfID uint) error {
	if err := db.Delete(&PortForward{}, pfID).Error; err != nil {
		logger.Error("Failed to delete port forward", "pfID", pfID, "error", err)
		return err
	}
	return nil
}

func CountPortForwards() (int64, error) {
	var count int64
	if err := db.Model(&PortForward{}).Count(&count).Error; err != nil {
		logger.Error("Failed to count port forwards", "error", err)
		return 0, err
	}
	return count, nil
}
