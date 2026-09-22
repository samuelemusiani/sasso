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
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.Use(&ErrorMetricsPlugin{})
	if err != nil {
		return fmt.Errorf("failed to initialize error metrics plugin: %w", err)
	}

	err = initGlobals()
	if err != nil {
		return fmt.Errorf("failed to initialize globals: %w", err)
	}

	err = initRealms()
	if err != nil {
		return fmt.Errorf("failed to initialize realms: %w", err)
	}

	err = initBackupRequests()
	if err != nil {
		return fmt.Errorf("failed to initialize backup requests: %w", err)
	}

	err = initNotifications()
	if err != nil {
		return fmt.Errorf("failed to initialize notifications: %w", err)
	}

	err = initGroupResources()
	if err != nil {
		return fmt.Errorf("failed to initialize group resources: %w", err)
	}

	err = initWireguardPeers()
	if err != nil {
		return fmt.Errorf("failed to initialize wireguard config: %w", err)
	}

	err = initVMExpirationNotifications()
	if err != nil {
		return fmt.Errorf("failed to initialize VM expiration notifications: %w", err)
	}

	err = initGroups()
	if err != nil {
		return fmt.Errorf("failed to initialize groups: %w", err)
	}

	err = initSettings()
	if err != nil {
		return fmt.Errorf("failed to initialize settings: %w", err)
	}

	err = initUsers()
	if err != nil {
		return fmt.Errorf("failed to initialize users: %w", err)
	}

	err = initVMs()
	if err != nil {
		return fmt.Errorf("failed to initialize VMs: %w", err)
	}

	err = initPortForwards()
	if err != nil {
		return fmt.Errorf("failed to initialize port forwards: %w", err)
	}

	err = initNetworks()
	if err != nil {
		return fmt.Errorf("failed to initialize networks: %w", err)
	}

	err = initInterfaces()
	if err != nil {
		return fmt.Errorf("failed to initialize interfaces: %w", err)
	}

	err = initSSHKeys()
	if err != nil {
		return fmt.Errorf("failed to initialize ssh keys: %w", err)
	}

	err = initTelegramBots()
	if err != nil {
		return fmt.Errorf("failed to initialize telegram bots: %w", err)
	}

	err = applyFixes()
	if err != nil {
		return fmt.Errorf("failed to apply fixes to database: %w", err)
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
		return fmt.Errorf("failed to migrate globals table: %w", err)
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
		return fmt.Errorf("failed to update database version: %w", err)
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
			return fmt.Errorf("failed to get admin user ID during fixes application: %w", err)
		}

		var adminSettings Setting

		err = tx.Where(&Setting{UserID: adminID}).First(&adminSettings).Error
		if err == nil {
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to find admin user settings during fixes application: %w", err)
		}

		logger.Info("Admin user has no settings, creating default settings", "userID", adminID)

		err = createDefaultSettingsForUserTransaction(tx, adminID)
		if err != nil {
			return fmt.Errorf("failed to create default settings for admin user during fixes application: %w", err)
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
