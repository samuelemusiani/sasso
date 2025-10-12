package db

import (
	"time"

	"gorm.io/gorm"
)

type Group struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name        string
	Description string
}

type UserGroup struct {
	UserID    uint `gorm:"primaryKey"`
	GroupID   uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Role      string // e.g., "member", "admin", "owner"
}

func initGroups() error {
	return db.AutoMigrate(&Group{}, &UserGroup{})
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
	err := db.Joins("JOIN user_groups ON user_groups.group_id = groups.id").
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
		if err := tx.Delete(&Group{}, groupID).Error; err != nil {
			logger.Error("Failed to delete group", "error", err)
			return err
		}
		return nil
	})
}

func GetGroupByID(groupID uint) (*Group, error) {
	var group Group
	err := db.First(&group, groupID).Error
	if err != nil {
		logger.Error("Failed to retrieve group by ID", "error", err)
		return nil, err
	}
	return &group, nil
}

func GetUserRoleInGroup(userID, groupID uint) (string, error) {
	var userGroup UserGroup
	err := db.First(&userGroup, "user_id = ? AND group_id = ?", userID, groupID).Error
	if err != nil {
		logger.Error("Failed to retrieve user role in group", "error", err)
		return "", err
	}
	return userGroup.Role, nil
}
