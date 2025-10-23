package db

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"samuelemusiani/sasso/vpn/config"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

var (
	db     *gorm.DB     = nil
	logger *slog.Logger = nil

	ErrAlreadyExists = fmt.Errorf("record already exists")
)

func Init(l *slog.Logger, c *config.Database) error {
	logger = l
	var err error

	url := fmt.Sprintf("host=%s user=%s password=%s dbname=sasso port=%d sslmode=disable", c.Host, c.User, c.Password, c.Port)

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
