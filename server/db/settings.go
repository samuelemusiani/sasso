package db

import "gorm.io/gorm"

type Setting struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;uniqueIndex:idx_user_key"`

	// Notification preferences
	MailPortForwardNotification          bool `gorm:"not null;default:true"`
	MailVMStatusUpdateNotification       bool `gorm:"not null;default:true"`
	MailGlobalSSHKeysChangeNotification  bool `gorm:"not null;default:true"`
	MailVMExpirationNotification         bool `gorm:"not null;default:true"`
	MailVMEliminatedNotification         bool `gorm:"not null;default:true"`
	MailVMStoppedNotification            bool `gorm:"not null;default:true"`
	MailSSHKeysChangedOnVM               bool `gorm:"not null;default:true"`
	MailUserInvitation                   bool `gorm:"not null;default:true"`
	MailUserRemovalFromGroupNotification bool `gorm:"not null;default:true"`

	TelegramPortForwardNotification          bool `gorm:"not null;default:true"`
	TelegramVMStatusUpdateNotification       bool `gorm:"not null;default:true"`
	TelegramGlobalSSHKeysChangeNotification  bool `gorm:"not null;default:true"`
	TelegramVMExpirationNotification         bool `gorm:"not null;default:true"`
	TelegramVMEliminatedNotification         bool `gorm:"not null;default:true"`
	TelegramVMStoppedNotification            bool `gorm:"not null;default:true"`
	TelegramSSHKeysChangedOnVM               bool `gorm:"not null;default:true"`
	TelegramUserInvitation                   bool `gorm:"not null;default:true"`
	TelegramUserRemovalFromGroupNotification bool `gorm:"not null;default:true"`
}

func initSettings() error {
	if err := db.AutoMigrate(&Setting{}); err != nil {
		logger.Error("Failed to migrate settings table", "error", err)
		return err
	}
	logger.Debug("Settings table migrated successfully")
	return nil
}

func GetSettingsByUserID(userID uint) (*Setting, error) {
	var setting Setting
	if err := db.Where(&Setting{UserID: userID}).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.Error("Failed to get settings by user ID", "userID", userID, "error", err)
		return nil, err
	}
	return &setting, nil
}

func CreateDefaultSettingsForUser(userID uint) error {
	setting := Setting{
		UserID: userID,
		// Default notification preferences
		MailPortForwardNotification:          true,
		MailVMStatusUpdateNotification:       true,
		MailGlobalSSHKeysChangeNotification:  true,
		MailVMExpirationNotification:         true,
		MailVMEliminatedNotification:         true,
		MailVMStoppedNotification:            true,
		MailSSHKeysChangedOnVM:               true,
		MailUserInvitation:                   true,
		MailUserRemovalFromGroupNotification: true,

		TelegramPortForwardNotification:          true,
		TelegramVMStatusUpdateNotification:       true,
		TelegramGlobalSSHKeysChangeNotification:  true,
		TelegramVMExpirationNotification:         true,
		TelegramVMEliminatedNotification:         true,
		TelegramVMStoppedNotification:            true,
		TelegramSSHKeysChangedOnVM:               true,
		TelegramUserInvitation:                   true,
		TelegramUserRemovalFromGroupNotification: true,
	}

	if err := db.Create(&setting).Error; err != nil {
		logger.Error("Failed to create default settings for user", "userID", userID, "error", err)
		return err
	}
	return nil
}

func UpdateSettings(setting *Setting) error {
	if err := db.Save(setting).Error; err != nil {
		logger.Error("Failed to update settings", "error", err)
		return err
	}
	return nil
}
