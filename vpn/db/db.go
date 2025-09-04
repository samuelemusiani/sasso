package db

import (
	// "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
)

var (
	db *gorm.DB = nil
)

func InitDB() error {
	// TODO: connect to the database
	if err := initInterfaces(); err != nil {
		slog.Error("Error initializing interfaces:", err)
		return err
	}
	return nil
}
