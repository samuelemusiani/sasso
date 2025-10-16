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

	Status string `gorm:"type:varchar(20);not null;default:'unknown';check:status IN ('unknown','pending','ready','reconfiguring','creating','deleting','pre-creating','pre-deleting')"`

	OwnerID   uint   `gorm:"not null;index"`
	OwnerType string `gorm:"not null;index"`

	PortForwards []PortForward `gorm:"foreignKey:VNetID;constraint:OnDelete:CASCADE"`
}

func initNetworks() error {
	if err := db.AutoMigrate(&Net{}); err != nil {
		logger.Error("Failed to migrate networks table", "error", err)
		return err
	}
	logger.Debug("Networks table migrated successfully")
	return nil
}

func GetNetByID(ID uint) (*Net, error) {
	var net Net
	if err := db.First(&net, ID).Error; err != nil {
		logger.Error("Failed to find network by ID", "netID", ID, "error", err)
		return nil, err
	}
	return &net, nil
}

func GetNetByName(name string) (*Net, error) {
	var net Net
	if err := db.Where("name = ?", name).First(&net).Error; err != nil {
		logger.Error("Failed to find network by name", "netName", name, "error", err)
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
		logger.Error("Failed to get random available tag by zone", "zone", zone, "error", err)
		return 0, err
	}
	return uint32(tag), nil
}

func GetNetsByUserID(userID uint) ([]Net, error) {
	var nets []Net
	if err := db.Where("owner_id = ? AND owner_type = ?", userID, "User").Find(&nets).Error; err != nil {
		logger.Error("Failed to get nets for user", "userID", userID, "error", err)
		return nil, err
	}
	return nets, nil
}

func GetNetsByGroupID(groupID uint) ([]Net, error) {
	var nets []Net
	if err := db.Where("owner_id = ? AND owner_type = ?", groupID, "Group").Find(&nets).Error; err != nil {
		logger.Error("Failed to get nets for group", "groupID", groupID, "error", err)
		return nil, err
	}
	return nets, nil
}

// Only counts nets owned by users, not groups
func CountNetsByUserID(userID uint) (uint, error) {
	var count int64
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ?", userID, "User").Count(&count).Error; err != nil {
		logger.Error("Failed to count nets for user", "userID", userID, "error", err)
		return 0, err
	}
	return uint(count), nil
}

// Only counts nets owned by groups, not users
func CountNetsByGroupID(groupID uint) (uint, error) {
	var count int64
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ?", groupID, "Group").Count(&count).Error; err != nil {
		logger.Error("Failed to count nets for group", "groupID", groupID, "error", err)
		return 0, err
	}
	return uint(count), nil
}

func GetSubnetsByUserID(userID uint) ([]string, error) {
	var subnets []string
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ? AND status = ?", userID, "User", "ready").Pluck("subnet", &subnets).Error; err != nil {
		logger.Error("Failed to get subnets for user", "userID", userID, "error", err)
		return nil, err
	}
	return subnets, nil
}

func GetSubnetsByGroupID(groupID uint) ([]string, error) {
	var subnets []string
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ? AND status = ?", groupID, "Group", "ready").Pluck("subnet", &subnets).Error; err != nil {
		logger.Error("Failed to get subnets for group", "groupID", groupID, "error", err)
		return nil, err
	}
	return subnets, nil
}

func IsAddressAGatewayOrBroadcast(address string) (bool, error) {
	var count int64

	addressLike := address + "/%"

	if err := db.Model(&Net{}).Where("gateway LIKE ? OR broadcast LIKE ?", addressLike, addressLike).Count(&count).Error; err != nil {
		logger.Error("Failed to check if address is a gateway or broadcast", "address", address, "error", err)
		return false, err
	}
	return count > 0, nil
}

// This function only creates a network for a user in the DB. It does
// not create the network in Proxmox
func CreateNetForUser(userID uint, name, alias, zone string, tag uint32, vlanAware bool, status string) (*Net, error) {

	net := &Net{
		Name:      string(name[:]),
		Alias:     alias,
		Zone:      zone,
		Tag:       tag,
		VlanAware: vlanAware,
		OwnerID:   userID,
		OwnerType: "User",
		Status:    status,
	}

	if err := db.Create(net).Error; err != nil {
		logger.Error("Failed to create network for user", "userID", userID, "error", err)
		return nil, err
	}

	logger.Debug("Created network for user", "userID", userID, "netName", net.Name, "zone", net.Zone, "tag", net.Tag, "vlanAware", net.VlanAware)

	return net, nil
}

// This function only creates a network for a group in the DB. It does
// not create the network in Proxmox
func CreateNetForGroup(groupID uint, name, alias, zone string, tag uint32, vlanAware bool, status string) (*Net, error) {

	net := &Net{
		Name:      string(name[:]),
		Alias:     alias,
		Zone:      zone,
		Tag:       tag,
		VlanAware: vlanAware,
		OwnerID:   groupID,
		OwnerType: "Group",
		Status:    status,
	}

	if err := db.Create(net).Error; err != nil {
		logger.Error("Failed to create network for group", "groupID", groupID, "error", err)
		return nil, err
	}

	logger.Debug("Created network for group", "groupID", groupID, "netName", net.Name, "zone", net.Zone, "tag", net.Tag, "vlanAware", net.VlanAware)

	return net, nil
}

func GetVNetsWithStatus(status string) ([]Net, error) {
	var nets []Net
	if err := db.Where("status = ?", status).Find(&nets).Error; err != nil {
		logger.Error("Failed to get VNets with status", "status", status, "error", err)
		return nil, err
	}
	return nets, nil
}

func UpdateVNetStatus(ID uint, status string) error {
	if err := db.Model(&Net{}).Where("id = ?", ID).Update("status", status).Error; err != nil {
		logger.Error("Failed to update VNet status", "netID", ID, "status", status, "error", err)
		return err
	}
	logger.Debug("Updated VNet status", "netID", ID, "status", status)
	return nil
}

func DeleteNetByID(ID uint) error {
	if err := db.Delete(&Net{}, ID).Error; err != nil {
		logger.Error("Failed to delete network", "netID", ID, "error", err)
		return err
	}
	logger.Debug("Deleted network", "netID", ID)
	return nil
}

func UpdateVNet(net *Net) error {
	if err := db.Save(net).Error; err != nil {
		logger.Error("Failed to update network", "netID", net.ID, "error", err)
		return err
	}
	logger.Debug("Updated network", "netID", net.ID)
	return nil
}

func GetAllNets() ([]Net, error) {
	var nets []Net
	if err := db.Find(&nets).Error; err != nil {
		logger.Error("Failed to get all VNets", "error", err)
		return nil, err
	}
	return nets, nil
}

func GetVNetBySubnet(subnet string) (*Net, error) {
	var net Net
	if err := db.Where("subnet = ?", subnet).First(&net).Error; err != nil {
		logger.Error("Failed to find network by subnet", "subnet", subnet, "error", err)
		return nil, err
	}
	return &net, nil
}

func CountVNets() (int64, error) {
	var count int64
	if err := db.Model(&Net{}).Count(&count).Error; err != nil {
		logger.Error("Failed to count VNets", "error", err)
		return 0, err
	}
	return count, nil
}
