package db

import "gorm.io/gorm"

type Ticket struct {
	UUID        string `gorm:"primaryKey"`
	RequestType string `gorm:"not null"`
}

type NetworkRequest struct {
	Ticket
	VNet   string `gorm:"not null"` // Name of the new VNet
	VNetID uint   `gorm:"not null"` // ID of the new VNet (VXLAN ID)

	Status  string `gorm:"not null;default:'pending'"` // Status of the request
	Success bool   `gorm:"not null;default:false"`     // True if the request was successful
	Error   string `gorm:"not null;default:''"`        // Error message if the request failed

	Subnet    string `gorm:"not null"` // Subnet of the new VNet
	RouterIP  string `gorm:"not null"` // Router IP of the new VNet
	Broadcast string `gorm:"not null"` // Broadcast address of the new VNet
}

func initTickets() error {
	return db.AutoMigrate(&Ticket{}, &NetworkRequest{})
}

func GetTicketByUUID(uuid string) (*Ticket, error) {
	var t Ticket
	if err := db.First(&t, "uuid = ?", uuid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.With("error", err).Error("Failed to retrieve ticket by UUID")
		return nil, err
	}

	return &t, nil
}

func GetNetworkRequestByTicket(t *Ticket) (*NetworkRequest, error) {
	var req NetworkRequest
	if err := db.First(&req, "uuid = ?", t.UUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.With("error", err).Error("Failed to retrieve network request by ticket")
		return nil, err
	}

	return &req, nil
}

func SaveNetworkRequest(req NetworkRequest) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&req.Ticket).Error; err != nil {
			logger.With("error", err).Error("Failed to create network request")
			return err
		}

		if err := tx.Create(&req).Error; err != nil {
			logger.With("error", err).Error("Failed to create network request details")
			return err
		}

		return nil
	})
}
