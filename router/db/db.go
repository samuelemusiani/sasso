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
	"samuelemusiani/sasso/router/config"
)

var (
	db     *gorm.DB
	logger *slog.Logger

	ErrNotFound = errors.New("record not found")
)

func Init(dbLogger *slog.Logger, c config.Database) error {
	err := checkConfig(c)
	if err != nil {
		return err
	}

	logger = dbLogger

	url := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", c.Host, c.User, c.Password, c.Database, c.Port)

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

	err = initInterfaces()
	if err != nil {
		logger.Error("Failed to initialize subnets in database", "error", err)

		return err
	}

	err = initPortForwards()
	if err != nil {
		logger.Error("Failed to initialize port forwards in database", "error", err)

		return err
	}

	return nil
}

func checkConfig(c config.Database) error {
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
