package db

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"samuelemusiani/sasso/server/config"
)

var (
	db     *gorm.DB
	logger *slog.Logger

	ErrNotFound              = errors.New("record not found")
	ErrForbidden             = errors.New("forbidden")
	ErrAlreadyExists         = errors.New("record already exists")
	ErrInsufficientResources = errors.New("insufficient resources")
	ErrResourcesInUse        = errors.New("resources are in use")
)

func Init(dbLogger *slog.Logger, c config.Database) error {
	logger = dbLogger

	if err := checkConfig(&c); err != nil {
		return err
	}

	var err error

	url := fmt.Sprintf("host=%s user=%s password=%s dbname=sasso port=%d sslmode=disable", c.Host, c.User, c.Password, c.Port)

	logger.Debug("Connecting to database", "url", url)

	db, err = gorm.Open(postgres.Open(url), &gorm.Config{
		Logger: gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  gormlogger.Error,
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

	err = initSettings()
	if err != nil {
		logger.Error("Failed to initialize settings in database", "error", err)

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

	err = applyFixes()
	if err != nil {
		logger.Error("Failed to apply fixes to database", "error", err)

		return err
	}

	return nil
}

type Globals struct {
	gorm.Model

	Version string
}

func initGlobals() error {
	err := db.AutoMigrate(&Globals{})
	if err != nil {
		logger.Error("Failed to migrate Globals table", "error", err)

		return err
	}

	var globals Globals
	db.First(&globals)

	currentVersion := "0.0.1"

	if globals.Version == currentVersion {
		return nil
	}

	logger.Info("Database version mismatch", "old", globals.Version, "current", currentVersion)
	globals.Version = currentVersion

	err = db.Save(&globals).Error
	if err != nil {
		logger.Error("Failed to update database version", "error", err)

		return err
	}

	return nil
}

// This functions applies necessary fixes to the database. Most fixes are
// necessary because of bugs in previous versions of the software. One could
// update the database by hand, but this function automates the process.
func applyFixes() error {
	err := db.Transaction(func(tx *gorm.DB) error {
		// After 6feb102b98a0c60516bf506c4e7a07b4f8cca750, the admin User is being
		// created with the CreateUser function and has default settings created.
		// We check if the admin user has settings, and if not, we create them.
		adminID, err := getAdminIDTransaction(tx)
		if err != nil {
			logger.Error("Failed to get admin user ID during fixes application", "error", err)

			return err
		}

		var adminSettings Setting

		err = tx.Where(&Setting{UserID: adminID}).First(&adminSettings).Error
		if err == nil {
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Failed to find admin user settings during fixes application", "error", err)

			return err
		}

		logger.Info("Admin user has no settings, creating default settings", "userID", adminID)

		err = createDefaultSettingsForUserTransaction(tx, adminID)
		if err != nil {
			logger.Error("Failed to create default settings for admin user during fixes application", "error", err)

			return err
		}

		return nil
	})

	return err
}

func checkConfig(c *config.Database) error {
	if c.User == "" {
		return errors.New("database user is empty")
	}

	if c.Password == "" {
		return errors.New("database password is empty")
	}

	if c.Database == "" {
		return errors.New("database name is empty")
	}

	if c.Host == "" {
		return errors.New("database host is empty")
	}

	if c.Port == 0 {
		return errors.New("database port is empty")
	}

	return nil
}
