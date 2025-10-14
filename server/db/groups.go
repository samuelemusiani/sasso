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
}

type UserGroup struct {
	UserID    uint `gorm:"primaryKey"`
	GroupID   uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Role      string // e.g., "member", "admin", "owner"
}

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
		if err := tx.Delete(&UserGroup{}, "group_id = ?", groupID).Error; err != nil {
			logger.Error("Failed to delete user-group associations", "error", err)
			return err
		}
		result := tx.Delete(&Group{}, groupID)
		if result.Error != nil {
			logger.Error("Failed to delete group", "error", result.Error)
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
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
	result := db.Where("user_id = ? AND group_id = ?", userID, groupID).
		Delete(&UserGroup{})
	if result.Error != nil {
		logger.Error("Failed to remove user from group", "error", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
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
