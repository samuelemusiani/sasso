package db

import "time"

type Notification struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Status  string `gorm:"not null;default:'pending','sent'"`
	Body    string `gorm:"type:text;not null"`
	Subject string `gorm:"type:varchar(255);not null"`

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
	if err := db.Where("status = ?", "pending").Find(&notifs).Error; err != nil {
		logger.Error("Failed to get pending notifications", "error", err)
		return nil, err
	}
	return notifs, nil
}

func SetNotificationAsSent(id uint) error {
	if err := db.Model(&Notification{}).Where("id = ?", id).Update("status", "sent").Error; err != nil {
		logger.Error("Failed to set notification as sent", "id", id, "error", err)
		return err
	}
	return nil
}
