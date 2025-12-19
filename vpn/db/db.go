package db

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
	"samuelemusiani/sasso/vpn/config"
)

var (
	db     *gorm.DB     = nil
	logger *slog.Logger = nil

	ErrAlreadyExists = fmt.Errorf("record already exists")
)

func Init(l *slog.Logger, c *config.Database) error {
	err := checkConfig(c)
	if err != nil {
		return err
	}

	logger = l

	url := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", c.Host, c.User, c.Password, c.Database, c.Port)

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

	if err := initSubnets(); err != nil {
		logger.Error("Failed to initialize subnets in database", "error", err)
		return err
	}

	if err := initPeers(); err != nil {
		logger.Error("Failed to initialize peers in database", "error", err)
		return err
	}

	return nil
}

func checkConfig(c *config.Database) error {
	if c.User == "" {
		return fmt.Errorf("database user is empty")
	}

	if c.Password == "" {
		return fmt.Errorf("database password is empty")
	}

	if c.Database == "" {
		return fmt.Errorf("database name is empty")
	}

	if c.Host == "" {
		return fmt.Errorf("database host is empty")
	}

	if c.Port == 0 {
		return fmt.Errorf("database port is empty")
	}

	return nil
}
