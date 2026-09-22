package db

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	LocalRealmType = "local"
	LDAPRealmType  = "ldap"
)

type Realm struct {
	gorm.Model

	Name        string `gorm:"uniqueIndex;not null"`
	Description string `gorm:"not null"`
	Type        string `gorm:"not null;default:'local'"`

	Users []User `gorm:"foreignKey:RealmID"`
}

type LDAPRealm struct {
	Realm `gorm:"embedded;embeddedPrefix:realm_"`

	URL        string `gorm:"not null"`
	UserBaseDN string `gorm:"not null"`
	BindDN     string `gorm:"not null"`
	Password   string `gorm:"not null"`

	LoginFilter       string `gorm:"not null"`
	MaintainerGroupDN string `gorm:"not null"`
	AdminGroupDN      string `gorm:"not null"`

	MailAttribute string `gorm:"not null;default:'mail'"`
}

func initRealms() error {
	err := db.AutoMigrate(&Realm{}, &LDAPRealm{})
	if err != nil {
		return fmt.Errorf("failed to migrate realms table: %w", err)
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
		return fmt.Errorf("failed to create local realm: %w", result.Error)
	}

	logger.Debug("Local realm initialized successfully")

	return nil
}

func GetAllRealms() ([]Realm, error) {
	var realms []Realm

	result := db.Find(&realms)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve realms: %w", result.Error)
	}

	return realms, nil
}

func AddLDAPRealm(realm LDAPRealm) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&realm.Realm).Error; err != nil {
			return fmt.Errorf("failed to create associated realm for LDAP realm: %w", err)
		}

		if err := tx.Create(&realm).Error; err != nil {
			return fmt.Errorf("failed to add LDAP realm: %w", err)
		}

		logger.Debug("LDAP realm added successfully")

		return nil
	})
}

func GetRealmByID(id uint) (*Realm, error) {
	var realm Realm
	if err := db.First(&realm, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to find realm by ID: %w", err)
	}

	return &realm, nil
}

func GetLDAPRealmByID(id uint) (*LDAPRealm, error) {
	var ldapRealm LDAPRealm
	if err := db.First(&ldapRealm, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to find LDAP realm by ID: %w", err)
	}

	return &ldapRealm, nil
}

func DeleteRealmByID(id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&Realm{}, "id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to delete realm: %w", err)
		}

		if err := tx.Delete(&LDAPRealm{}, "realm_id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to delete associated LDAP realm: %w", err)
		}

		logger.Debug("Realm deleted successfully", "realmID", id)

		return nil
	})
}

func UpdateLDAPRealm(realm LDAPRealm) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&realm.Realm).Error; err != nil {
			return fmt.Errorf("failed to update associated realm for LDAP realm: %w", err)
		}

		if err := tx.Save(&realm).Error; err != nil {
			return fmt.Errorf("failed to update LDAP realm: %w", err)
		}

		logger.Debug("LDAP realm updated successfully", "realmID", realm.ID)

		return nil
	})
}

func GetRealmByName(name string) (*Realm, error) {
	var realm Realm
	if err := db.First(&realm, "name = ?", name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to find realm by name: %w", err)
	}

	return &realm, nil
}
