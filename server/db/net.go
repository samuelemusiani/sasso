package db

import "fmt"

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
	Interfaces   []Interface   `gorm:"foreignKey:VNetID;constraint:OnDelete:CASCADE"`
}

func initNetworks() error {
	if err := db.AutoMigrate(&Net{}); err != nil {
		return fmt.Errorf("failed to migrate networks table: %w", err)
	}

	logger.Debug("Networks table migrated successfully")

	return nil
}

func GetNetByID(id uint) (*Net, error) {
	var net Net
	if err := db.First(&net, id).Error; err != nil {
		return nil, fmt.Errorf("failed to find network by ID: %w", err)
	}

	return &net, nil
}

func GetNetByName(name string) (*Net, error) {
	var net Net
	if err := db.Where("name = ?", name).First(&net).Error; err != nil {
		return nil, fmt.Errorf("failed to find network by name: %w", err)
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
		return 0, fmt.Errorf("failed to get random available tag by zone: %w", err)
	}

	return uint32(tag), nil
}

func GetNetsByUserID(userID uint) ([]Net, error) {
	var nets []Net
	if err := db.Where("owner_id = ? AND owner_type = ?", userID, "User").Find(&nets).Error; err != nil {
		return nil, fmt.Errorf("failed to get nets for user: %w", err)
	}

	return nets, nil
}

func GetNetsByGroupID(groupID uint) ([]Net, error) {
	var nets []Net
	if err := db.Where("owner_id = ? AND owner_type = ?", groupID, "Group").Find(&nets).Error; err != nil {
		return nil, fmt.Errorf("failed to get nets for group: %w", err)
	}

	return nets, nil
}

// CountNetsByUserID only counts nets owned by users, not groups
func CountNetsByUserID(userID uint) (uint, error) {
	var count int64
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ?", userID, "User").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count nets for user: %w", err)
	}

	return uint(count), nil
}

// CountNetsByGroupID only counts nets owned by groups, not users
func CountNetsByGroupID(groupID uint) (uint, error) {
	var count int64
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ?", groupID, "Group").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count nets for group: %w", err)
	}

	return uint(count), nil
}

func GetSubnetsByUserID(userID uint) ([]string, error) {
	var subnets []string
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ? AND status = ?", userID, "User", "ready").Pluck("subnet", &subnets).Error; err != nil {
		return nil, fmt.Errorf("failed to get subnets for user: %w", err)
	}

	return subnets, nil
}

func GetSubnetsFromGroupsWhereUserIsAdminOrOwner(userID uint) ([]string, error) {
	var subnets []string

	err := db.Table("nets").
		Joins("JOIN user_groups ug ON nets.owner_id = ug.group_id AND nets.owner_type = ?", "Group").
		Where("ug.user_id = ? AND (ug.role = ? OR ug.role = ?)", userID, "admin", "owner").
		Pluck("nets.subnet", &subnets).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get subnets from groups where user is admin or owner: %w", err)
	}

	return subnets, nil
}

func GetSubnetsByGroupID(groupID uint) ([]string, error) {
	var subnets []string
	if err := db.Model(&Net{}).Where("owner_id = ? AND owner_type = ? AND status = ?", groupID, "Group", "ready").Pluck("subnet", &subnets).Error; err != nil {
		return nil, fmt.Errorf("failed to get subnets for group: %w", err)
	}

	return subnets, nil
}

func IsAddressAGatewayOrBroadcast(address string) (bool, error) {
	var count int64

	addressLike := address + "/%"

	if err := db.Model(&Net{}).Where("gateway LIKE ? OR broadcast LIKE ?", addressLike, addressLike).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check if address is a gateway or broadcast: %w", err)
	}

	return count > 0, nil
}

// CreateNetForUser only creates a network for a user in the DB. It does
// not create the network in Proxmox
func CreateNetForUser(userID uint, name, alias, zone string, tag uint32, vlanAware bool, status string) (*Net, error) {
	net := &Net{
		Name:      name,
		Alias:     alias,
		Zone:      zone,
		Tag:       tag,
		VlanAware: vlanAware,
		OwnerID:   userID,
		OwnerType: "User",
		Status:    status,
	}

	if err := db.Create(net).Error; err != nil {
		return nil, fmt.Errorf("failed to create network for user: %w", err)
	}

	logger.Debug("Created network for user", "userID", userID, "netName", net.Name, "zone", net.Zone, "tag", net.Tag, "vlanAware", net.VlanAware)

	return net, nil
}

// CreateNetForGroup only creates a network for a group in the DB. It does
// not create the network in Proxmox
func CreateNetForGroup(groupID uint, name, alias, zone string, tag uint32, vlanAware bool, status string) (*Net, error) {
	net := &Net{
		Name:      name,
		Alias:     alias,
		Zone:      zone,
		Tag:       tag,
		VlanAware: vlanAware,
		OwnerID:   groupID,
		OwnerType: "Group",
		Status:    status,
	}

	if err := db.Create(net).Error; err != nil {
		return nil, fmt.Errorf("failed to create network for group: %w", err)
	}

	logger.Debug("Created network for group", "groupID", groupID, "netName", net.Name, "zone", net.Zone, "tag", net.Tag, "vlanAware", net.VlanAware)

	return net, nil
}

func GetVNetsWithStatus(status string) ([]Net, error) {
	var nets []Net
	if err := db.Where("status = ?", status).Find(&nets).Error; err != nil {
		return nil, fmt.Errorf("failed to get VNets with status: %w", err)
	}

	return nets, nil
}

func UpdateVNetStatus(id uint, status string) error {
	if err := db.Model(&Net{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update VNet status: %w", err)
	}

	logger.Debug("Updated VNet status", "netID", id, "status", status)

	return nil
}

func DeleteNetByID(id uint) error {
	if err := db.Delete(&Net{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	logger.Debug("Deleted network", "netID", id)

	return nil
}

func UpdateVNet(net *Net) error {
	if err := db.Save(net).Error; err != nil {
		return fmt.Errorf("failed to update network: %w", err)
	}

	logger.Debug("Updated network", "netID", net.ID)

	return nil
}

func UpdateVNetName(id uint, newName string) error {
	err := db.Model(&Net{}).Where("id = ?", id).
		UpdateColumn("name", newName).
		Error
	if err != nil {
		return fmt.Errorf("failed to update VNet name: %w", err)
	}

	logger.Debug("Updated VNet name", "netID", id, "newName", newName)

	return nil
}

func GetAllNets() ([]Net, error) {
	var nets []Net
	if err := db.Find(&nets).Error; err != nil {
		return nil, fmt.Errorf("failed to get all VNets: %w", err)
	}

	return nets, nil
}

func GetVNetBySubnet(subnet string) (*Net, error) {
	var net Net
	if err := db.Where("subnet = ?", subnet).First(&net).Error; err != nil {
		return nil, fmt.Errorf("failed to find network by subnet: %w", err)
	}

	return &net, nil
}

func CountNetsWithStates() ([]StatusCount, error) {
	var counts []StatusCount

	result := db.Model(&Net{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&counts)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count nets with states: %w", result.Error)
	}

	return counts, nil
}
