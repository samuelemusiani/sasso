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

	ErrNotFound              = errors.New("record not found")
	ErrForbidden             = errors.New("forbidden")
	ErrAlreadyExists         = errors.New("record already exists")
	ErrInsufficientResources = errors.New("insufficient resources")
	ErrResourcesInUse        = errors.New("resources are in use")
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
		logger.Error("Failed to connect to database", "error", err)
		return err
	}

	err = initGlobals()
	if err != nil {
		logger.Error("Failed to set globals in database", "error", err)
		return err
	}

	err = initRealms()
	if err != nil {
		logger.Error("Failed to initialize realms in database", "error", err)
		return err
	}

	err = initBackupRequests()
	if err != nil {
		logger.Error("Failed to initialize backup requests in database", "error", err)
		return err
	}

	err = initNotifications()
	if err != nil {
		logger.Error("Failed to initialize notifications in database", "error", err)
		return err
	}

	err = initGroupResources()
	if err != nil {
		logger.Error("Failed to initialize group resources in database", "error", err)
		return err
	}

	err = initVPNConfig()
	if err != nil {
		logger.Error("Failed to initialize VPN config in database", "error", err)
		return err
	}

	err = initVMExpirationNotifications()
	if err != nil {
		logger.Error("Failed to initialize VM expiration notifications in database", "error", err)
		return err
	}

	err = initGroups()
	if err != nil {
		logger.Error("Failed to initialize groups in database", "error", err)
		return err
	}

	err = initUsers()
	if err != nil {
		logger.Error("Failed to initialize users in database", "error", err)
		return err
	}

	err = initVMs()
	if err != nil {
		logger.Error("Failed to initialize VMs in database", "error", err)
		return err
	}

	err = initPortForwards()
	if err != nil {
		logger.Error("Failed to initialize port forwards in database", "error", err)
		return err
	}

	err = initNetworks()
	if err != nil {
		logger.Error("Failed to initialize networks in database", "error", err)
		return err
	}

	err = initInterfaces()
	if err != nil {
		logger.Error("Failed to initialize interfaces in database", "error", err)
		return err
	}

	err = initSSHKeys()
	if err != nil {
		logger.Error("Failed to initialize ssh keys in database", "error", err)
		return err
	}

	err = initTelegramBots()
	if err != nil {
		logger.Error("Failed to initialize telegram bots in database", "error", err)
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

	logger.Info("Database version mismatch", "old", globals.Version, "current", currentVersion)
	globals.Version = currentVersion
	err := db.Save(&globals).Error
	if err != nil {
		logger.Error("Failed to update database version", "error", err)
		return err
	}

	return nil
}
