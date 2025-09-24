package db

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"samuelemusiani/sasso/server/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

var (
	db     *gorm.DB     = nil
	logger *slog.Logger = nil

	ErrNotFound = errors.New("record not found")
)

func Init(dbLogger *slog.Logger, c config.Database) error {
	logger = dbLogger

	var err error

	url := fmt.Sprintf("host=%s user=%s password=%s dbname=sasso port=%d sslmode=disable", c.Host, c.User, c.Password, c.Port)

	logger.Debug("Connecting to database", "url", url)

	db, err = gorm.Open(postgres.Open(url), &gorm.Config{
		Logger: gorm_logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gorm_logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  gorm_logger.Error,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		logger.With("error", err).Error("Failed to connect to database")
		return err
	}

	err = initGlobals()
	if err != nil {
		logger.With("error", err).Error("Failed to set globals in database")
		return err
	}

	err = initRealms()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize realms in database")
		return err
	}

	err = initUsers()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize users in database")
		return err
	}

	err = initVMs()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize VMs in database")
		return err
	}

	err = initPortForwards()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize port forwards in database")
		return err
	}

	err = initNetworks()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize networks in database")
		return err
	}

	err = initInterfaces()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize interfaces in database")
		return err
	}

	err = initSSHKeys()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize ssh keys in database")
		return err
	}

	return nil
}

type Globals struct {
	gorm.Model
	Version string
}

func initGlobals() error {
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
