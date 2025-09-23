package db

import "gorm.io/gorm"

type PortForward struct {
	gorm.Model

	OutPort  uint16 `gorm:"not null; uniqueIndex"`
	DestPort uint16 `gorm:"not null"`
	DestIP   string `gorm:"not null"`
	UserID   uint   `gorm:"not null"`
	Approved bool   `gorm:"not null;default:false"`
}

func initPortForwards() error {
	if err := db.AutoMigrate(&PortForward{}); err != nil {
		logger.With("error", err).Error("Failed to migrate port forwards table")
		return err
	}
	logger.Info("Port forwards table migrated successfully")
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

func AddPortForward(outPort uint16, destPort uint16, destIP string, userID uint) (*PortForward, error) {
	pf := &PortForward{
		OutPort:  outPort,
		DestPort: destPort,
		DestIP:   destIP,
		UserID:   userID,
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
