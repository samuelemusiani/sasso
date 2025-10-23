package db

import (
	"time"
)

type VPNConfig struct {
	ID        uint `gorm:"primaryKey"`
	UpdatedAt time.Time
}
