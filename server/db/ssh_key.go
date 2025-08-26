package db

import "time"

type SSHKey struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name   string `gorm:"type:varchar(255);not null"`
	Key    string `gorm:"type:text;not null"`
	UserID uint   `gorm:"not null"`
	Global bool   `gorm:"default:false"`
}

func initSSHKeys() error {
	err := db.AutoMigrate(&SSHKey{})
	if err != nil {
		logger.With("error", err).Error("Failed to migrate SSHKeys table")
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
	result := db.Create(sshKey)
	if result.Error != nil {
		return nil, result.Error
	}
	return sshKey, nil
}

func CreateGlobalSSHKey(name string, key string) (*SSHKey, error) {
	admin, err := GetUserByUsername("admin")
	if err != nil {
		return nil, err
	}
	sshKey := &SSHKey{
		Name:   name,
		Key:    key,
		Global: true,
		UserID: admin.ID,
	}
	result := db.Create(sshKey)
	if result.Error != nil {
		return nil, result.Error
	}
	return sshKey, nil
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
