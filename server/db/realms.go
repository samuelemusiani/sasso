package db

import "gorm.io/gorm"

type Realm struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex;not null"`
	Description string `gorm:"not null"`
	Type        string `gorm:"not null;default:'local'"`
}

type LDAPRealm struct {
	Realm    `gorm:"embedded;embeddedPrefix:realm_"`
	URL      string `gorm:"not null"`
	BaseDN   string `gorm:"not null"`
	BindDN   string `gorm:"not null"`
	Password string `gorm:"not null"`
}

func initRealms() error {
	err := db.AutoMigrate(&Realm{}, &LDAPRealm{})
	if err != nil {
		logger.With("error", err).Error("Failed to migrate Realms table")
		return err
	}

	var localRealm Realm
	result := db.First(&localRealm, "name = ?", "Local")
	if result.Error == nil {
		logger.Debug("Local realm already exists")
		return nil
	}

	localRealm = Realm{
		Name:        "Local",
		Description: "Local authentication realm",
		Type:        "local",
	}
	result = db.Create(&localRealm)
	if result.Error != nil {
		logger.With("error", result.Error).Error("Failed to create local realm")
		return result.Error
	}

	logger.Info("Local realm initialized successfully")
	return nil
}

func GetAllRealms() ([]Realm, error) {
	var realms []Realm
	result := db.Find(&realms)
	if result.Error != nil {
		logger.With("error", result.Error).Error("Failed to retrieve realms")
		return nil, result.Error
	}
	return realms, nil
}
