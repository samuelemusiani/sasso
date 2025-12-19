package db

import "gorm.io/gorm"

type Setting struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;uniqueIndex:idx_user_key;constraint:OnDelete:CASCADE"`

	// Notification preferences
	MailPortForwardNotification          bool `gorm:"not null;default:true"`
	MailVMStatusUpdateNotification       bool `gorm:"not null;default:true"`
	MailGlobalSSHKeysChangeNotification  bool `gorm:"not null;default:true"`
	MailVMExpirationNotification         bool `gorm:"not null;default:true"`
	MailVMEliminatedNotification         bool `gorm:"not null;default:true"`
	MailVMStoppedNotification            bool `gorm:"not null;default:true"`
	MailSSHKeysChangedOnVMNotification   bool `gorm:"not null;default:true"`
	MailUserInvitationNotification       bool `gorm:"not null;default:true"`
	MailUserRemovalFromGroupNotification bool `gorm:"not null;default:true"`
	MailLifetimeOfVMExpiredNotification  bool `gorm:"not null;default:true"`

	TelegramPortForwardNotification          bool `gorm:"not null;default:true"`
	TelegramVMStatusUpdateNotification       bool `gorm:"not null;default:true"`
	TelegramGlobalSSHKeysChangeNotification  bool `gorm:"not null;default:true"`
	TelegramVMExpirationNotification         bool `gorm:"not null;default:true"`
	TelegramVMEliminatedNotification         bool `gorm:"not null;default:true"`
	TelegramVMStoppedNotification            bool `gorm:"not null;default:true"`
	TelegramSSHKeysChangedOnVMNotification   bool `gorm:"not null;default:true"`
	TelegramUserInvitationNotification       bool `gorm:"not null;default:true"`
	TelegramUserRemovalFromGroupNotification bool `gorm:"not null;default:true"`
	TelegramLifetimeOfVMExpiredNotification  bool `gorm:"not null;default:true"`
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

func createDefaultSettingsForUserTransaction(tx *gorm.DB, userID uint) error {
	setting := Setting{
		UserID: userID,
		// Default notification preferences
		MailPortForwardNotification:          true,
		MailVMStatusUpdateNotification:       true,
		MailGlobalSSHKeysChangeNotification:  true,
		MailVMExpirationNotification:         true,
		MailVMEliminatedNotification:         true,
		MailVMStoppedNotification:            true,
		MailSSHKeysChangedOnVMNotification:   true,
		MailUserInvitationNotification:       true,
		MailUserRemovalFromGroupNotification: true,
		MailLifetimeOfVMExpiredNotification:  true,

		TelegramPortForwardNotification:          true,
		TelegramVMStatusUpdateNotification:       true,
		TelegramGlobalSSHKeysChangeNotification:  true,
		TelegramVMExpirationNotification:         true,
		TelegramVMEliminatedNotification:         true,
		TelegramVMStoppedNotification:            true,
		TelegramSSHKeysChangedOnVMNotification:   true,
		TelegramUserInvitationNotification:       true,
		TelegramUserRemovalFromGroupNotification: true,
		TelegramLifetimeOfVMExpiredNotification:  true,
	}

	if err := tx.Create(&setting).Error; err != nil {
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
