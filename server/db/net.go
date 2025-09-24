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

	UserID uint   `gorm:"not null"`
	Status string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('unknown','pending','ready','creating','deleting','pre-creating','pre-deleting')"`

	PortForwards []PortForward `gorm:"foreignKey:VNetID;constraint:OnDelete:CASCADE"`
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

func GetSubnetsByUserID(userID uint) ([]string, error) {
	var subnets []string
	if err := db.Model(&Net{}).Where("user_id = ? AND status = ?", userID, "ready").Pluck("subnet", &subnets).Error; err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to get subnets for user")
		return nil, err
	}
	return subnets, nil
}

func IsAddressAGatewayOrBroadcast(address string) (bool, error) {
	var count int64
	if err := db.Model(&Net{}).Where("gateway = ? OR broadcast = ?", address, address).Count(&count).Error; err != nil {
		logger.With("address", address, "error", err).Error("Failed to check if address is a gateway or broadcast")
		return false, err
	}
	return count > 0, nil
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

func GetAllNets() ([]Net, error) {
	var nets []Net
	if err := db.Find(&nets).Error; err != nil {
		logger.With("error", err).Error("Failed to get all VNets")
		return nil, err
	}
	return nets, nil
}

func GetVNetBySubnet(subnet string) (*Net, error) {
	var net Net
	if err := db.Where("subnet = ?", subnet).First(&net).Error; err != nil {
		logger.With("subnet", subnet, "error", err).Error("Failed to find network by subnet")
		return nil, err
	}
	return &net, nil
}
