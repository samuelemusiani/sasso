package db

import (
	"time"

	"gorm.io/gorm"
)

type Notification struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Status  string `gorm:"not null;default:'pending','sent'"`
	Body    string `gorm:"type:text;not null"`
	Subject string `gorm:"type:varchar(255);not null"`

	Email    bool `gorm:"not null"`
	Telegram bool `gorm:"not null"`

	UserID uint `gorm:"not null"`
}

func initNotifications() error {
	if err := db.AutoMigrate(&Notification{}); err != nil {
		logger.Error("Failed to migrate notifications table", "error", err)
		return err
	}

	logger.Debug("Notifications table migrated successfully")

	return nil
}

func GetPendingNotifications() ([]Notification, error) {
	var notifs []Notification
	if err := db.Where(&Notification{Status: "pending"}).Find(&notifs).Error; err != nil {
		logger.Error("Failed to get pending notifications", "error", err)
		return nil, err
	}

	return notifs, nil
}

func SetNotificationAsSent(id uint) error {
	if err := db.Model(&Notification{ID: id}).Update("status", "sent").Error; err != nil {
		logger.Error("Failed to set notification as sent", "id", id, "error", err)
		return err
	}

	return nil
}

func InsertNotification(userID uint, subject, body string, mail, telegram bool) error {
	ntf := Notification{
		UserID:   userID,
		Subject:  subject,
		Body:     body,
		Email:    mail,
		Telegram: telegram,
		Status:   "pending",
	}

	if err := db.Create(&ntf).Error; err != nil {
		logger.Error("Failed to insert notification", "error", err)
		return err
	}

	return nil
}

type TelegramBot struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"type:varchar(255);not null"`
	Notes     string `gorm:"type:text"`
	Token     string `gorm:"type:varchar(255);not null"`
	ChatID    string `gorm:"type:varchar(255);not null"`
	UserID    uint   `gorm:"not null"`
	Enabled   bool   `gorm:"not null;default:true"`
}

func initTelegramBots() error {
	if err := db.AutoMigrate(&TelegramBot{}); err != nil {
		logger.Error("Failed to migrate telegram_bots table", "error", err)
		return err
	}

	logger.Debug("Telegram bots table migrated successfully")

	return nil
}

func GetTelegramBotsByUserID(userID uint) ([]TelegramBot, error) {
	var bots []TelegramBot
	if err := db.Where("user_id = ?", userID).Find(&bots).Error; err != nil {
		logger.Error("Failed to get telegram bots by user ID", "userID", userID, "error", err)
		return nil, err
	}

	return bots, nil
}

func GetEnabledTelegramBotsByUserID(userID uint) ([]TelegramBot, error) {
	var bots []TelegramBot
	if err := db.Where("user_id = ? AND enabled = ?", userID, true).Find(&bots).Error; err != nil {
		logger.Error("Failed to get enabled telegram bots by user ID", "userID", userID, "error", err)
		return nil, err
	}

	return bots, nil
}

func CreateTelegramBot(name, notes, token, chatID string, userID uint) error {
	bot := &TelegramBot{
		Name:   name,
		Notes:  notes,
		Token:  token,
		ChatID: chatID,
		UserID: userID,
	}
	if err := db.Create(bot).Error; err != nil {
		logger.Error("Failed to create telegram bot", "error", err)
		return err
	}

	return nil
}

func DeleteTelegramBot(id uint, userID uint) error {
	err := db.Where("id = ? AND user_id = ?", id, userID).Delete(&TelegramBot{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}

		logger.Error("Failed to delete telegram bot", "id", id, "userID", userID, "error", err)

		return err
	}

	return nil
}

func GetUsersWithTelegramBots() ([]uint, error) {
	var userIDs []uint
	if err := db.Model(&TelegramBot{}).Where("enabled = ?", true).Distinct().Pluck("user_id", &userIDs).Error; err != nil {
		logger.Error("Failed to get users with telegram bots", "error", err)
		return nil, err
	}

	return userIDs, nil
}

func GetTelegramBotByID(id uint) (*TelegramBot, error) {
	var bot TelegramBot
	if err := db.Where("id = ?", id).First(&bot).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}

		logger.Error("Failed to get telegram bot by ID", "id", id, "error", err)

		return nil, err
	}

	return &bot, nil
}

func ChangeTelegramBotEnabled(id uint, userID uint, enabled bool) error {
	if err := db.Model(&TelegramBot{}).Where("id = ? AND user_id = ?", id, userID).Update("enabled", enabled).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}

		logger.Error("Failed to change telegram bot enabled status", "id", id, "enabled", enabled, "error", err)

		return err
	}

	return nil
}
