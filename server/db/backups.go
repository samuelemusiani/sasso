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

	OwnerID   uint   `gorm:"not null;index"`
	OwnerType string `gorm:"not null;index"`

	// For creation
	Name  string `gorm:"not null"`
	Notes string `gorm:"not null"`
}

func initBackupRequests() error {
	if err := db.AutoMigrate(&BackupRequest{}); err != nil {
		logger.Error("Failed to migrate backup_requests table", "error", err)
		return err
	}
	return nil
}

func GetBackupRequestByID(id uint) (*BackupRequest, error) {
	var backupRequest BackupRequest
	result := db.First(&backupRequest, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	return &backupRequest, nil
}

func NewBackupRequestForUser(backupType, status string, vmID, userID uint, name, notes string) (*BackupRequest, error) {
	return NewBackupRequestWithVolidForUser(backupType, status, nil, vmID, userID, name, notes)
}

func NewBackupRequestForGroup(backupType, status string, vmID, groupID uint, name, notes string) (*BackupRequest, error) {
	return NewBackupRequestWithVolidForGroup(backupType, status, nil, vmID, groupID, name, notes)
}

func NewBackupRequestWithVolidForUser(backupType, status string, volid *string, vmID, userID uint, name, notes string) (*BackupRequest, error) {
	return newBackupRequestWithVolid(backupType, status, volid, vmID, userID, "User", name, notes)
}

func NewBackupRequestWithVolidForGroup(backupType, status string, volid *string, vmID, groupID uint, name, notes string) (*BackupRequest, error) {
	return newBackupRequestWithVolid(backupType, status, volid, vmID, groupID, "Group", name, notes)
}

func newBackupRequestWithVolid(backupType, status string, volid *string, vmID, ownerID uint, ownerType, name, notes string) (*BackupRequest, error) {
	backupRequest := &BackupRequest{
		Type:      backupType,
		Status:    status,
		VMID:      vmID,
		Volid:     volid,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		Name:      name,
		Notes:     notes,
	}
	result := db.Create(backupRequest)
	if result.Error != nil {
		return nil, result.Error
	}
	return backupRequest, nil
}

func UpdateBackupRequestStatus(id uint, status string) error {
	result := db.Model(&BackupRequest{ID: id}).Update("status", status)
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
	result := db.Where(&BackupRequest{Status: status, Type: t}).Find(&backupRequests)
	if result.Error != nil {
		return nil, result.Error
	}
	return backupRequests, nil
}

func GetBackupRequestsByUserID(userID uint) ([]BackupRequest, error) {
	return getBackupRequestsByOwnerID(userID, "User")
}

func GetBackupRequestsByGroupID(groupID uint) ([]BackupRequest, error) {
	return getBackupRequestsByOwnerID(groupID, "Group")
}

func getBackupRequestsByOwnerID(ownerID uint, ownerType string) ([]BackupRequest, error) {
	var backupRequests []BackupRequest
	result := db.Where(&BackupRequest{OwnerID: ownerID, OwnerType: ownerType}).
		Find(&backupRequests)
	if result.Error != nil {
		return nil, result.Error
	}
	return backupRequests, nil
}

func IsAPendingBackupRequest(vmID uint) (bool, error) {
	var count int64
	result := db.Model(&BackupRequest{}).
		Where(&BackupRequest{ID: vmID, Type: "pending"}).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func IsAPendingBackupRequestWithVolid(vmID uint, volid string) (bool, error) {
	var count int64
	result := db.Model(&BackupRequest{}).
		Where(&BackupRequest{ID: vmID, Volid: &volid, Type: "pending"}).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func GetBackupRequestsByVMIDStatusAndType(vmID uint, status, t string) ([]BackupRequest, error) {
	var backupRequests []BackupRequest
	result := db.Where(&BackupRequest{VMID: vmID, Status: status, Type: t}).Find(&backupRequests)
	if result.Error != nil {
		return nil, result.Error
	}
	return backupRequests, nil
}
