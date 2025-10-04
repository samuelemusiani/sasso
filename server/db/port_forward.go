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
	UserID   uint   `gorm:"not null"`
	Approved bool   `gorm:"not null;default:false"`

	VNetID uint `gorm:"not null"`
}

type PortForwardWithUsername struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	OutPort  uint16 `gorm:"not null; uniqueIndex"`
	DestPort uint16 `gorm:"not null"`
	DestIP   string `gorm:"not null"`
	UserID   uint   `gorm:"not null"`
	Approved bool   `gorm:"not null;default:false"`
	VNetID   uint   `gorm:"not null"`
	Username string
}

func initPortForwards() error {
	if err := db.AutoMigrate(&PortForward{}); err != nil {
		logger.With("error", err).Error("Failed to migrate port forwards table")
		return err
	}
	logger.Debug("Port forwards table migrated successfully")
	return nil
}

func GetPortForwards() ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Find(&pfs).Error; err != nil {
		logger.With("error", err).Error("Failed to get all port forwards")
		return nil, err
	}
	return pfs, nil
}

func GetPortForwardsWithUsernames() ([]PortForwardWithUsername, error) {
	var pfs []PortForwardWithUsername
	if err := db.Table("port_forwards pf").Select("pf.*, u.username").
		Joins("left join users u on pf.user_id = u.id").
		Scan(&pfs).Error; err != nil {
		logger.With("error", err).Error("Failed to get port forwards with usernames")
		return nil, err
	}
	return pfs, nil
}

func GetApprovedPortForwards() ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Where("approved = ?", true).Find(&pfs).Error; err != nil {
		logger.With("error", err).Error("Failed to get approved port forwards")
		return nil, err
	}
	return pfs, nil
}

func GetPortForwardByID(ID uint) (*PortForward, error) {
	var pf PortForward
	if err := db.First(&pf, ID).Error; err != nil {
		logger.With("pfID", ID, "error", err).Error("Failed to find port forward by ID")
		return nil, err
	}
	return &pf, nil
}

func GetPortForwardsByUserID(userID uint) ([]PortForward, error) {
	var pfs []PortForward
	if err := db.Where("user_id = ?", userID).Find(&pfs).Error; err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get port forwards for user")
		return nil, err
	}
	return pfs, nil
}

func AddPortForward(outPort, destPort uint16, destIP, subnet string, userID uint) (*PortForward, error) {

	net, err := GetVNetBySubnet(subnet)
	if err != nil {
		logger.With("subnet", subnet, "error", err).Error("Failed to find VNet by subnet")
		return nil, err
	}

	pf := &PortForward{
		OutPort:  outPort,
		DestPort: destPort,
		DestIP:   destIP,
		UserID:   userID,
		Approved: false,
		VNetID:   net.ID,
	}
	if err := db.Create(pf).Error; err != nil {
		logger.With("error", err).Error("Failed to create port forward")
		return nil, err
	}
	return pf, nil
}

func UpdatePortForwardApproval(pfID uint, approve bool) error {
	if err := db.Model(&PortForward{}).Where("id = ?", pfID).Update("approved", approve).Error; err != nil {
		logger.With("pfID", pfID, "error", err).Error("Failed to update port forward approval")
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
		logger.With("error", err).Error("Failed to get random available out port")
		return 0, err
	}
	return uint16(outPort), nil
}

func DeletePortForward(pfID uint) error {
	if err := db.Delete(&PortForward{}, pfID).Error; err != nil {
		logger.With("pfID", pfID, "error", err).Error("Failed to delete port forward")
		return err
	}
	return nil
}

func CountPortForwards() (int64, error) {
	var count int64
	if err := db.Model(&PortForward{}).Count(&count).Error; err != nil {
		logger.With("error", err).Error("Failed to count port forwards")
		return 0, err
	}
	return count, nil
}
