package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Group struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name        string
	Description string

	// Read-only field populated via join
	// Role of the user that is querying the groups
	Role string `gorm:"->"`

	VMs            []VM            `gorm:"polymorphic:Owner;polymorphicValue:Group"`
	Nets           []Net           `gorm:"polymorphic:Owner;polymorphicValue:Group"`
	PortForwards   []PortForward   `gorm:"polymorphic:Owner;polymorphicValue:Group"`
	BackupRequests []BackupRequest `gorm:"polymorphic:Owner;polymorphicValue:Group"`

	Users     []User          `gorm:"many2many:user_groups;"`
	Resources []GroupResource `gorm:"foreignKey:GroupID"`
}

type UserGroup struct {
	UserID    uint `gorm:"primaryKey"`
	GroupID   uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Role      string // e.g., "member", "admin", "owner"
}

var lastUserGroupTableUpdate time.Time = time.Time{}

func (ug *UserGroup) AfterUpdate(tx *gorm.DB) (err error) {
	lastUserGroupTableUpdate = time.Now()
	return nil
}

func (ug *UserGroup) AfterCreate(tx *gorm.DB) (err error) {
	lastUserGroupTableUpdate = time.Now()
	return nil
}

func (ug *UserGroup) AfterDelete(tx *gorm.DB) (err error) {
	lastUserGroupTableUpdate = time.Now()
	return nil
}

func GetLastUserGroupUpdate() time.Time {
	return lastUserGroupTableUpdate
}

func UpdateGroupByID(groupID uint, name, description string) error {
	result := db.Model(&Group{}).Where("id = ?", groupID).
		Updates(Group{Name: name, Description: description})
	if result.Error != nil {
		logger.Error("Failed to update group", "error", result.Error)
		return result.Error
	}
	return nil
}

// This struct is only used for queries
type GroupMemberWithUsername struct {
	UserID   uint
	Username string
	Role     string
}

type GroupInvitation struct {
	ID      uint `gorm:"primaryKey"`
	GroupID uint
	UserID  uint
	Role    string
	State   string // e.g., "pending", "accepted", "declined"

	Username         string `gorm:"->;-:migration"`
	GroupName        string `gorm:"->;-:migration"`
	GroupDescription string `gorm:"->;-:migration"`
}

func initGroups() error {
	return db.AutoMigrate(&Group{}, &UserGroup{}, &GroupInvitation{})
}

func CreateGroup(name, description string, userID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		group := Group{
			Name:        name,
			Description: description,
		}
		if err := tx.Create(&group).Error; err != nil {
			logger.Error("Failed to create group", "error", err)
			return err
		}

		userGroup := UserGroup{
			UserID:  userID,
			GroupID: group.ID,
			Role:    "owner",
		}
		if err := tx.Create(&userGroup).Error; err != nil {
			logger.Error("Failed to add user to group", "error", err)
			return err
		}

		res := GroupResource{
			GroupID: group.ID,
			UserID:  1,
			Nets:    1, // Allocate 1 net from admin to the group by default
		}
		if err := tx.Create(&res).Error; err != nil {
			logger.Error("Failed to allocate default resources to group", "error", err)
			return err
		}
		return nil
	})
}

func GetGroupsByUserID(userID uint) ([]Group, error) {
	var groups []Group
	err := db.Table("groups").
		Select("groups.*, user_groups.role as role").
		Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ?", userID).
		Find(&groups).Error
	if err != nil {
		logger.Error("Failed to retrieve groups by user ID", "error", err)
		return nil, err
	}
	return groups, nil
}

func DeleteGroup(groupID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Delete all invitations for the group
		if err := tx.Delete(&GroupInvitation{}, "group_id = ?", groupID).Error; err != nil {
			logger.Error("Failed to delete group invitations", "error", err)
			return err
		}

		// Remove every user from the group
		if err := tx.Delete(&UserGroup{}, "group_id = ?", groupID).Error; err != nil {
			logger.Error("Failed to delete user-group associations", "error", err)
			return err
		}

		// Find all group resources and return them to users
		var resources []GroupResource
		err := tx.Model(&GroupResource{}).
			Where("group_id = ?", groupID).
			Find(&resources).Error
		if err != nil {
			logger.Error("Failed to retrieve group resources", "error", err)
			return err
		}

		for _, r := range resources {
			if r.UserID == 1 {
				// Admin user, skip
				continue
			}
			err = tx.Model(&User{Model: gorm.Model{ID: r.UserID}}).
				UpdateColumns(map[string]interface{}{
					"max_cores": gorm.Expr("max_cores + ?", r.Cores),
					"max_ram":   gorm.Expr("max_ram + ?", r.RAM),
					"max_disk":  gorm.Expr("max_disk + ?", r.Disk),
				}).Error
			if err != nil {
				logger.Error("Failed to return resources to user", "error", err)
				return err
			}
		}
		err = tx.Where("group_id = ?", groupID).Delete(&GroupResource{}).Error

		// Delete the group
		result := tx.Delete(&Group{}, groupID)
		if result.Error != nil {
			logger.Error("Failed to delete group", "error", result.Error)
			return result.Error
		}

		return nil
	})
}

func GetGroupByID(groupID uint) (*Group, error) {
	var group Group
	err := db.First(&group, groupID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.Error("Failed to retrieve group by ID", "error", err)
		return nil, err
	}
	return &group, nil
}

func GetUserRoleInGroup(userID, groupID uint) (string, error) {
	var userGroup UserGroup
	err := db.First(&userGroup, "user_id = ? AND group_id = ?", userID, groupID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrNotFound
		}
		logger.Error("Failed to retrieve user role in group", "error", err)
		return "", err
	}
	return userGroup.Role, nil
}

func GetGroupMembers(groupID uint) ([]GroupMemberWithUsername, error) {
	var members []GroupMemberWithUsername
	err := db.Table("user_groups").
		Joins("JOIN users ON users.id = user_groups.user_id").
		Where("user_groups.group_id = ?", groupID).
		Select("users.id as user_id, users.username, user_groups.role").
		Scan(&members).Error
	if err != nil {
		logger.Error("Failed to retrieve group members", "error", err)
		return nil, err
	}

	return members, nil
}

// This functions is used to get pending invitations for a user along with group
// details
func GetGroupsWithInvitationByUserID(userID uint) ([]GroupInvitation, error) {
	var invitations []GroupInvitation
	err := db.Table("group_invitations as gi").
		Joins("JOIN groups ON groups.id = gi.group_id JOIN users ON users.id = gi.user_id").
		Select("gi.id, gi.group_id, groups.name as group_name, groups.description as group_description, gi.role, gi.state, users.username as username").
		Where("gi.user_id = ? AND state = ?", userID, "pending").
		Scan(&invitations).Error

	if err != nil {
		logger.Error("Failed to retrieve group invitations", "error", err)
		return nil, err
	}

	return invitations, nil
}

func DeclineGroupInvitation(invitationID, userID uint) error {
	err := db.Model(&GroupInvitation{}).Where("user_id = ? AND id = ? AND state = ?", userID, invitationID, "pending").
		Update("state", "declined").Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // No pending invitation found, nothing to do
		}
		logger.Error("Failed to decline invitation", "error", err)
		return err
	}
	return nil
}

func AcceptGroupInvitation(invitationID, userID uint) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		var invitation GroupInvitation
		err := tx.Where("user_id = ? AND id = ? AND state = ?", userID, invitationID, "pending").First(&invitation).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrNotFound
			}
			logger.Error("Failed to find invitation", "error", err)
			return err
		}

		err = tx.Model(&invitation).Update("state", "accepted").Error
		userGroup := UserGroup{
			UserID:  userID,
			GroupID: invitation.GroupID,
			Role:    invitation.Role,
		}
		err = tx.Create(&userGroup).Error
		if err != nil {
			logger.Error("Failed to update invitation state", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		logger.Error("Failed to accept invitation", "error", err)
		return err
	}
	return nil
}

// This function is used to get pending invitations for a group along with
// user details
func GetPendingGroupInvitationsByGroupID(groupID uint) ([]GroupInvitation, error) {
	var invitations []GroupInvitation
	err := db.Table("group_invitations as gi").
		Joins("JOIN users ON users.id = gi.user_id").
		Select("gi.id, gi.group_id, users.username as username, gi.role, gi.state").
		Where("gi.group_id = ? AND gi.state = ?", groupID, "pending").
		Scan(&invitations).Error
	if err != nil {
		logger.Error("Failed to retrieve group invitations", "error", err)
		return nil, err
	}
	return invitations, nil
}

func InviteUserToGroup(userID, groupID uint, role string) error {
	invitation := GroupInvitation{
		UserID:  userID,
		GroupID: groupID,
		Role:    role,
		State:   "pending",
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var count int64
		err := tx.Model(&GroupInvitation{}).Where("user_id = ? AND group_id = ? AND state = ?", userID, groupID, "pending").Count(&count).Error
		if err != nil {
			logger.Error("Failed to check existing invitations", "error", err)
			return err
		}
		if count > 0 {
			return ErrAlreadyExists
		}

		err = db.Create(&invitation).Error
		if err != nil {
			logger.Error("Failed to create group invitation", "error", err)
			return err
		}
		return nil
	})
	return err
}

func RevokeGroupInvitationToUser(inviteID, groupID uint) error {
	result := db.Where("id = ? AND group_id = ? AND state = ?", inviteID, groupID, "pending").Delete(&GroupInvitation{})
	if result.Error != nil {
		logger.Error("Failed to revoke group invitation", "error", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func RemoveUserFromGroup(userID, groupID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		err := revokeGroupResourcesTransaction(tx, groupID, userID)
		if err != nil {
			logger.Error("Failed to revoke group resources", "error", err)
			return err
		}

		result := tx.Where("user_id = ? AND group_id = ?", userID, groupID).
			Delete(&UserGroup{})
		if result.Error != nil {
			logger.Error("Failed to remove user from group", "error", result.Error)
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func DoesUserBelongToGroup(userID, groupID uint) (bool, error) {
	var count int64
	err := db.Model(&UserGroup{}).Where("user_id = ? AND group_id = ?", userID, groupID).Count(&count).Error
	if err != nil {
		logger.Error("Failed to check user membership in group", "error", err)
		return false, err
	}
	return count > 0, nil
}

func CountGroupMembers(groupID uint) (int64, error) {
	var count int64
	err := db.Model(&UserGroup{}).Where("group_id = ?", groupID).Count(&count).Error
	if err != nil {
		logger.Error("Failed to count group members", "error", err)
		return 0, err
	}
	return count, nil
}

func GetUserIDsByGroupID(groupID uint) ([]uint, error) {
	var userIDs []uint
	err := db.Model(&UserGroup{}).Where("group_id = ?", groupID).Pluck("user_id", &userIDs).Error
	if err != nil {
		logger.Error("Failed to get user IDs by group ID", "error", err)
		return nil, err
	}
	return userIDs, nil
}

// It is possible to asign resources to a group by admins. We have model this
// as the admin user assigning some of their own resources to the group.
// As the admin does not have resources, we need to check to avoid reassigning
// them to it when revoking group resources.
type GroupResource struct {
	GroupID uint `gorm:"primaryKey"`
	UserID  uint `gorm:"primaryKey"`

	Cores uint `gorm:"not null"`
	RAM   uint `gorm:"not null"`
	Disk  uint `gorm:"not null"`
	Nets  uint `gorm:"not null"`

	Username string `gorm:"->;-:migration"`
}

func initGroupResources() error {
	return db.AutoMigrate(&GroupResource{})
}

func GetGroupResourceLimits(groupID uint) (uint, uint, uint, uint, error) {
	var res struct {
		Cores uint
		RAM   uint
		Disk  uint
		Nets  uint
	}

	err := db.Model(&GroupResource{}).
		Where("group_id = ?", groupID).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk, SUM(nets) as nets").
		Scan(&res).Error
	if err != nil {
		logger.Error("Failed to get group max resources", "error", err)
		return 0, 0, 0, 0, err
	}

	return res.Cores, res.RAM, res.Disk, res.Nets, nil
}

func GetGroupResourcesByGroupID(groupID uint) ([]GroupResource, error) {
	var resources []GroupResource
	err := db.Table("group_resources as gr").
		Joins("JOIN users ON users.id = gr.user_id").
		Select("gr.group_id, gr.user_id, gr.cores, gr.ram, gr.disk, gr.nets, users.username as username").
		Where("gr.group_id = ?", groupID).
		Scan(&resources).Error
	if err != nil {
		logger.Error("Failed to get group resources by group ID", "error", err)
		return nil, err
	}
	return resources, nil
}

func AddGroupResources(groupID, userID uint, cores, ram, disk, nets uint) error {
	groupResource := GroupResource{
		GroupID: groupID,
		UserID:  userID,
		Cores:   cores,
		RAM:     ram,
		Disk:    disk,
		Nets:    nets,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var used struct {
			Cores uint
			RAM   uint
			Disk  uint
		}
		err := tx.Model(&VM{}).
			Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk").
			Where(&VM{OwnerID: userID, OwnerType: "User"}).Scan(&used).Error
		if err != nil {
			logger.Error("Failed to get user VM resources", "error", err)
			return err
		}

		var usedNets int64
		err = tx.Model(&Net{}).
			Where(&Net{OwnerID: userID, OwnerType: "User"}).
			Count(&usedNets).Error
		if err != nil {
			logger.Error("Failed to get user Net resources", "error", err)
			return err
		}

		var u User
		err = tx.First(&u, userID).Error
		if err != nil {
			logger.Error("Failed to get user", "error", err)
			return err
		}
		if used.Cores+cores > u.MaxCores || used.RAM+ram > u.MaxRAM || used.Disk+disk > u.MaxDisk || uint(usedNets)+nets > u.MaxNets {
			return ErrInsufficientResources
		}

		err = tx.Create(&groupResource).Error
		if err != nil {
			logger.Error("Failed to create group resource", "error", err)
			return err
		}

		err = tx.Model(&User{Model: gorm.Model{ID: userID}}).
			UpdateColumns(map[string]interface{}{
				"max_cores": gorm.Expr("max_cores - ?", cores),
				"max_ram":   gorm.Expr("max_ram - ?", ram),
				"max_disk":  gorm.Expr("max_disk - ?", disk),
				"max_nets":  gorm.Expr("max_nets - ?", nets),
			}).Error
		if err != nil {
			logger.Error("Failed to update user limits", "error", err)
			return err
		}
		return nil
	})
	return err
}

func UpdateGroupResourceByAdmin(groupID, cores, ram, disk, nets uint) error {
	var groupResource GroupResource

	err := db.Where(&GroupResource{GroupID: groupID, UserID: 1}).
		First(&groupResource).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			groupResource = GroupResource{
				GroupID: groupID,
				UserID:  1, // Admin user
			}
		} else {
			logger.Error("Failed to check existing group resource by admin", "error", err)
			return err
		}
	}

	groupResource.Cores = cores
	groupResource.RAM = ram
	groupResource.Disk = disk
	groupResource.Nets = nets

	err = db.Save(&groupResource).Error
	if err != nil {
		logger.Error("Failed to create group resource by admin", "error", err)
		return err
	}
	return nil
}

func RevokeGroupResources(groupID, userID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return revokeGroupResourcesTransaction(tx, groupID, userID)
	})
}

func SetGroupResourcesByUserID(groupID, userID, cores, ram, disk, nets uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if userID == 1 {
			return nil
		}

		var resource GroupResource
		err := tx.Where(&GroupResource{GroupID: groupID, UserID: userID}).First(&resource).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			logger.Error("Failed to find group resource", "error", err)
			return err
		}

		used, maxResourceAvailable, err := getUsedAndMaxResourcesForGroupID(tx, groupID)
		if err != nil {
			logger.Error("Failed to get group resources", "error", err)
			return err
		}

		if maxResourceAvailable.Cores+(cores-resource.Cores) < used.Cores ||
			maxResourceAvailable.RAM+(ram-resource.RAM) < used.RAM ||
			maxResourceAvailable.Disk+(disk-resource.Disk) < used.Disk ||
			maxResourceAvailable.Nets+(nets-resource.Nets) < used.Nets {
			return ErrResourcesInUse
		}

		resource.Cores = cores
		resource.RAM = ram
		resource.Disk = disk
		resource.Nets = nets

		err = tx.Save(&resource).Error
		if err != nil {
			logger.Error("Failed to delete group resource", "error", err)
			return err
		}

		err = tx.Model(&User{Model: gorm.Model{ID: userID}}).
			UpdateColumns(map[string]interface{}{
				"max_cores": resource.Cores,
				"max_ram":   resource.RAM,
				"max_disk":  resource.Disk,
				"max_nets":  resource.Nets,
			}).Error
		if err != nil {
			logger.Error("Failed to update user limits", "error", err)
			return err
		}
		return nil
	})
}

func revokeGroupResourcesTransaction(tx *gorm.DB, groupID, userID uint) error {
	if userID == 1 {
		// Admin user, no resources to revoke
		return tx.Delete(&GroupResource{GroupID: groupID, UserID: userID}).Error
	}

	var resource GroupResource
	err := tx.Where(&GroupResource{GroupID: groupID, UserID: userID}).First(&resource).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No resources to revoke
			return nil
		}
		logger.Error("Failed to find group resource", "error", err)
		return err
	}

	used, maxResourceAvailable, err := getUsedAndMaxResourcesForGroupID(tx, groupID)
	if err != nil {
		logger.Error("Failed to get group resources", "error", err)
		return err
	}

	if maxResourceAvailable.Cores-resource.Cores < used.Cores ||
		maxResourceAvailable.RAM-resource.RAM < used.RAM ||
		maxResourceAvailable.Disk-resource.Disk < used.Disk ||
		maxResourceAvailable.Nets-resource.Nets < used.Nets {
		return ErrResourcesInUse
	}

	err = tx.Delete(&resource).Error
	if err != nil {
		logger.Error("Failed to delete group resource", "error", err)
		return err
	}

	err = tx.Model(&User{Model: gorm.Model{ID: userID}}).
		UpdateColumns(map[string]interface{}{
			"max_cores": gorm.Expr("max_cores + ?", resource.Cores),
			"max_ram":   gorm.Expr("max_ram + ?", resource.RAM),
			"max_disk":  gorm.Expr("max_disk + ?", resource.Disk),
			"max_nets":  gorm.Expr("max_nets + ?", resource.Nets),
		}).Error
	if err != nil {
		logger.Error("Failed to update user limits", "error", err)
		return err
	}
	return nil
}

type usedResources struct {
	Cores uint
	RAM   uint
	Disk  uint
	Nets  uint
}

func getUsedAndMaxResourcesForGroupID(tx *gorm.DB, groupID uint) (usedResources, usedResources, error) {
	var used usedResources
	err := tx.Model(&VM{}).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk").
		Where(&VM{OwnerID: groupID, OwnerType: "Group"}).Scan(&used).Error
	if err != nil {
		logger.Error("Failed to get group VM resources", "error", err)
		return usedResources{}, usedResources{}, err
	}

	var usedNets int64
	err = tx.Model(&Net{}).
		Where(&Net{OwnerID: groupID, OwnerType: "Group"}).
		Count(&usedNets).Error
	if err != nil {
		logger.Error("Failed to get group Net resources", "error", err)
		return usedResources{}, usedResources{}, err
	}
	used.Nets = uint(usedNets)

	var maxResource usedResources
	err = tx.Model(&GroupResource{}).
		Where("group_id = ?", groupID).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk, SUM(nets) as nets").
		Scan(&maxResource).Error
	if err != nil {
		logger.Error("Failed to get group max resources", "error", err)
		return usedResources{}, usedResources{}, err
	}

	return used, maxResource, nil
}

func GetAllGroups() ([]Group, error) {
	var groups []Group
	err := db.Find(&groups).Error
	if err != nil {
		logger.Error("Failed to retrieve all groups", "error", err)
		return nil, err
	}
	return groups, nil
}

func GetUserGroupResourcesByUserID(userID uint) (usedResources, error) {
	var res usedResources
	err := db.Model(&GroupResource{}).
		Where(&GroupResource{UserID: userID}).
		Select("SUM(cores) as cores, SUM(ram) as ram, SUM(disk) as disk, SUM(nets) as nets").
		Scan(&res).Error
	if err != nil {
		logger.Error("Failed to get user group resources by user ID", "error", err)
		return usedResources{}, err
	}
	return res, nil
}
