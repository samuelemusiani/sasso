package db

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"samuelemusiani/sasso/router/config"
	"time"

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

	url := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", c.Host, c.User, c.Password, c.Database, c.Port)

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

	err = initInterfaces()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize subnets in database")
		return err
	}

	err = initPortForwards()
	if err != nil {
		logger.With("error", err).Error("Failed to initialize port forwards in database")
		return err
	}

	return nil
}
