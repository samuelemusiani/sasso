package db

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type SSHKey struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name   string `gorm:"type:varchar(255);not null"`
	Key    string `gorm:"type:text;not null"`
	UserID uint   `gorm:"not null"`
	Global bool   `gorm:"default:false"`
}

var lastSSHKeyTableUpdate = time.Time{}

func (*SSHKey) AfterUpdate(_ *gorm.DB) (err error) {
	lastSSHKeyTableUpdate = time.Now()

	return nil
}

func (*SSHKey) AfterCreate(_ *gorm.DB) (err error) {
	lastSSHKeyTableUpdate = time.Now()

	return nil
}

func (*SSHKey) AfterDelete(_ *gorm.DB) (err error) {
	lastSSHKeyTableUpdate = time.Now()

	return nil
}

func GetLastSSHKeyUpdate() time.Time {
	return lastSSHKeyTableUpdate
}

func initSSHKeys() error {
	err := db.AutoMigrate(&SSHKey{})
	if err != nil {
		logger.Error("Failed to migrate SSHKeys table", "error", err)

		return err
	}

	return nil
}

func GetSSHKeysByUserID(userID uint) ([]SSHKey, error) {
	var keys []SSHKey

	result := db.Where("user_id = ?", userID).Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}

	return keys, nil
}

func GetGlobalSSHKeys() ([]SSHKey, error) {
	var keys []SSHKey

	result := db.Where("global = true").Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}

	return keys, nil
}

func CreateSSHKey(name string, key string, userID uint) (*SSHKey, error) {
	sshKey := &SSHKey{
		Name:   name,
		Key:    key,
		UserID: userID,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// We allow to add the same key for personal and global, but not to add
		// the same key twice for the same user in the same scope (personal or global)
		err := tx.Where(&SSHKey{Key: key, UserID: userID, Global: false}).
			First(&SSHKey{}).Error
		if err == nil {
			return ErrAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to check for existing SSH key: %w", err)
		}

		result := tx.Create(sshKey)
		if result.Error != nil {
			return result.Error
		}

		return nil
	})

	return sshKey, err
}

func CreateGlobalSSHKey(name string, key string) (*SSHKey, error) {
	admin, err := GetLocalAdmin()
	if err != nil {
		return nil, err
	}

	sshKey := &SSHKey{
		Name:   name,
		Key:    key,
		Global: true,
		UserID: admin.ID,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// We allow to add the same key for personal and global, but not to add
		// the same key twice for the same user in the same scope (personal or global)
		err := tx.Where(&SSHKey{Key: key, UserID: admin.ID, Global: true}).
			First(&SSHKey{}).Error
		if err == nil {
			return ErrAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to check for existing SSH key: %w", err)
		}

		result := tx.Create(sshKey)
		if result.Error != nil {
			return result.Error
		}

		return nil
	})

	return sshKey, err
}

func DeleteSSHKey(id uint, userID uint) error {
	result := db.Where("id = ? AND user_id = ?", id, userID).Delete(&SSHKey{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func DeleteGlobalSSHKey(id uint) error {
	result := db.Where("id = ? AND global = true", id).Delete(&SSHKey{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func GetSSHKeysByGroupID(groupID uint) ([]SSHKey, error) {
	var keys []SSHKey

	result := db.Table("ssh_keys").
		Select("ssh_keys.*").
		Joins("JOIN user_groups ON ssh_keys.user_id = user_groups.user_id").
		Where("user_groups.group_id = ?", groupID).
		Order("ssh_keys.id ASC").
		Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}

	return keys, nil
}
