package db

import (
	"fmt"
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
		return fmt.Errorf("failed to migrate port forwards table: %w", err)
	}

	logger.Debug("Port forwards table migrated successfully")

	return nil
}

func GetPortForwards() ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Find(&pfs).Error; err != nil {
		return nil, fmt.Errorf("failed to get all port forwards: %w", err)
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
		return nil, fmt.Errorf("failed to get port forwards with usernames: %w", err)
	}

	return portForwards, nil
}

func GetGroupPortForwardsByUserID(userID uint) ([]PortForward, error) {
	var pfs []PortForward

	err := db.Table("port_forwards pf").
		Select(`pf.*, groups.name as name`).
		Joins("JOIN groups ON pf.owner_type = ? AND pf.owner_id = groups.id", "Group").
		Joins("JOIN user_groups ug ON ug.group_id = groups.id").
		Where("ug.user_id = ?", userID).
		Find(&pfs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get group port forwards for user: %w", err)
	}

	return pfs, nil
}

func GetApprovedPortForwards() ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Where("approved = ?", true).Find(&pfs).Error; err != nil {
		return nil, fmt.Errorf("failed to get approved port forwards: %w", err)
	}

	return pfs, nil
}

func GetPortForwardByID(id uint) (*PortForward, error) {
	var pf PortForward
	if err := db.First(&pf, id).Error; err != nil {
		return nil, fmt.Errorf("failed to find port forward by ID: %w", err)
	}

	return &pf, nil
}

func GetPortForwardsByUserID(userID uint) ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Where(&PortForward{OwnerID: userID}).Find(&pfs).Error; err != nil {
		return nil, fmt.Errorf("failed to get port forwards for user: %w", err)
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
		return nil, fmt.Errorf("failed to find VNet by subnet: %w", err)
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
		return nil, fmt.Errorf("failed to create port forward: %w", err)
	}

	return pf, nil
}

func UpdatePortForwardApproval(pfID uint, approve bool) error {
	if err := db.Model(&PortForward{}).Where("id = ?", pfID).Update("approved", approve).Error; err != nil {
		return fmt.Errorf("failed to update port forward approval: %w", err)
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
		return 0, fmt.Errorf("failed to get random available out port: %w", err)
	}

	return uint16(outPort), nil
}

func DeletePortForward(pfID uint) error {
	if err := db.Delete(&PortForward{}, pfID).Error; err != nil {
		return fmt.Errorf("failed to delete port forward: %w", err)
	}

	return nil
}

func CountPortForwardsWithStates() ([]StatusCount, error) {
	var counts []StatusCount

	result := db.Model(&PortForward{}).
		Select("approved as status, COUNT(*) as count").
		Group("approved").
		Scan(&counts)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to count port forwards with states: %w", result.Error)
	}

	return counts, nil
}
