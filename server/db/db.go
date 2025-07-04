package db

import (
	"fmt"
	"log/slog"
	"samuelemusiani/sasso/server/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB = nil
var logger *slog.Logger = nil

func Init(dbLogger *slog.Logger, c config.Database) error {
	logger = dbLogger

	var err error

	url := fmt.Sprintf("host=%s user=%s password=%s dbname=sasso port=%d sslmode=disable", c.Host, c.User, c.Password, c.Port)

	db, err = gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		logger.With("error", err).Error("Failed to connect to database")
		return err
	}

	err = setGlobals()
	if err != nil {
		logger.With("error", err).Error("Failed to set globals in database")
		return err
	}

	return nil
}

func setGlobals() error {
	db.AutoMigrate(&Globals{})
	var globals Globals
	db.First(&globals)

	var currentVersion string = "0.0.1"

	if globals.Version == currentVersion {
		return nil
	}

	logger.With("old", globals.Version, "current", currentVersion).Info("Database version mismatch")
	globals.Version = currentVersion
	err := db.Save(&globals).Error
	if err != nil {
		logger.With("error", err).Error("Failed to update database version")
		return err
	}

	return nil
}

type Globals struct {
	gorm.Model
	Version string
}
