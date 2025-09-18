package db

type Net struct {
	ID        uint  `gorm:"primaryKey"`
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdatedAt int64 `gorm:"autoUpdateTime"`

	Name      string `gorm:"uniqueIndex;not null"`
	Alias     string `gorm:"not null"`             // For users
	Zone      string `gorm:"not null"`             // Should be "sasso"
	Tag       uint32 `gorm:"not null;uniqueIndex"` // Unique tag for the network
	VlanAware bool   `gorm:"not null;default:false"`

	Subnet    string `gorm:"not null"`
	Gateway   string `gorm:"not null"`
	Broadcast string `gorm:"not null"`

	UserID      uint    `gorm:"not null"`
	Status      string  `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('unknown','pending','ready','creating','deleting','pre-creating','pre-deleting')"`
	RouterTiket *string `gorm:"default:null"` // Ticket ID on the router side
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

func GetRandomAvailableTagByZone(zone string, start, end uint32) (uint32, error) {
	var tag int
	query := `
		SELECT n FROM generate_series(?::integer, ?::integer) AS n
		LEFT JOIN nets a ON a.tag = n AND a.zone = ?
		WHERE a.tag IS NULL
		ORDER BY RANDOM()
		LIMIT 1;
	`
	err := db.Raw(query, start, end, zone).Scan(&tag).Error
	if err != nil {
		logger.With("zone", zone, "error", err).Error("Failed to get random available tag by zone")
		return 0, err
	}
	return uint32(tag), nil
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

func UpdateVNet(net *Net) error {
	if err := db.Save(net).Error; err != nil {
		logger.With("netID", net.ID, "error", err).Error("Failed to update network")
		return err
	}
	logger.With("netID", net.ID).Info("Updated network")
	return nil
}

func AddTicketToNetByID(netID uint, ticketID string) error {
	if err := db.Model(&Net{}).Where("id = ?", netID).Update("router_tiket", ticketID).Error; err != nil {
		logger.With("netID", netID, "ticketID", ticketID, "error", err).Error("Failed to add ticket to network")
		return err
	}
	return nil
}

func GetTicketFromNetByID(netID uint) (*string, error) {
	var net Net
	if err := db.Select("router_tiket").First(&net, netID).Error; err != nil {
		logger.With("netID", netID, "error", err).Error("Failed to get ticket from network")
		return nil, err
	}
	return net.RouterTiket, nil
}
