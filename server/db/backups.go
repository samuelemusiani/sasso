package db

import (
	"time"

	"gorm.io/gorm"
)

type BackupRequest struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Type   string  `gorm:"type:varchar(20);not null;check:type IN ('create','restore','delete')"`
	Status string  `gorm:"type:varchar(20);not null;default:'pending';check:status IN ('pending','completed','failed')"`
	Volid  *string `gorm:"default:null"`
	VMID   uint    `gorm:"not null"`
	UserID uint    `gorm:"not null"`
}

func initBackupRequests() error {
	if err := db.AutoMigrate(&BackupRequest{}); err != nil {
		logger.With("error", err).Error("Failed to migrate backup_requests table")
		return err
	}
	return nil
}

func NewBackupRequest(backupType, status string, vmID, userID uint) (*BackupRequest, error) {
	return NewBackupRequestWithVolid(backupType, status, nil, vmID, userID)
}

func NewBackupRequestWithVolid(backupType, status string, volid *string, vmID, userID uint) (*BackupRequest, error) {
	backupRequest := &BackupRequest{
		Type:   backupType,
		Status: status,
		VMID:   vmID,
		Volid:  volid,
		UserID: userID,
	}
	result := db.Create(backupRequest)
	if result.Error != nil {
		return nil, result.Error
	}
	return backupRequest, nil
}

func UpdateBackupRequestStatus(id uint, status string) error {
	result := db.Model(&BackupRequest{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return ErrNotFound
		} else if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return result.Error
	}
	return nil
}

func GetBackupRequestWithStatusAndType(status, t string) ([]BackupRequest, error) {
	var backupRequests []BackupRequest
	result := db.Where("status = ? AND type = ?", status, t).Find(&backupRequests)
	if result.Error != nil {
		return nil, result.Error
	}
	return backupRequests, nil
}
