package db

import "gorm.io/gorm"

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
	Realm           `gorm:"embedded;embeddedPrefix:realm_"`
	URL             string `gorm:"not null"`
	UserBaseDN      string `gorm:"not null"`
	GroupBaseDN     string `gorm:"not null"`
	BindDN          string `gorm:"not null"`
	Password        string `gorm:"not null"`
	MaintainerGroup string `gorm:"not null"`
	AdminGroup      string `gorm:"not null"`
}

func initRealms() error {
	err := db.AutoMigrate(&Realm{}, &LDAPRealm{})
	if err != nil {
		logger.Error("Failed to migrate Realms table", "error", err)
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
		logger.Error("Failed to create local realm", "error", result.Error)
		return result.Error
	}

	logger.Debug("Local realm initialized successfully")
	return nil
}

func GetAllRealms() ([]Realm, error) {
	var realms []Realm
	result := db.Find(&realms)
	if result.Error != nil {
		logger.Error("Failed to retrieve realms", "error", result.Error)
		return nil, result.Error
	}
	return realms, nil
}

func AddLDAPRealm(realm LDAPRealm) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&realm.Realm).Error; err != nil {
			logger.Error("Failed to create associated Realm for LDAP realm", "error", err)
			return err
		}

		if err := tx.Create(&realm).Error; err != nil {
			logger.Error("Failed to add LDAP realm", "error", err)
			return err
		}

		logger.Debug("LDAP realm added successfully")
		return nil
	})
}

func GetRealmByID(id uint) (*Realm, error) {
	var realm Realm
	if err := db.First(&realm, id).Error; err != nil {
		logger.Error("Failed to find realm by ID", "realmID", id, "error", err)
		return nil, err
	}
	return &realm, nil
}

func GetLDAPRealmByID(id uint) (*LDAPRealm, error) {
	var ldapRealm LDAPRealm
	if err := db.First(&ldapRealm, id).Error; err != nil {
		logger.Error("Failed to find LDAP realm by ID", "ldapRealmID", id, "error", err)
		return nil, err
	}
	return &ldapRealm, nil
}

func DeleteRealmByID(id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&Realm{}, "id = ?", id).Error; err != nil {
			logger.Error("Failed to delete realm", "realmID", id, "error", err)
			return err
		}

		if err := tx.Delete(&LDAPRealm{}, "realm_id = ?", id).Error; err != nil {
			logger.Error("Failed to delete associated LDAP realm", "realmID", id, "error", err)
			return err
		}

		logger.Debug("Realm deleted successfully", "realmID", id)
		return nil
	})
}

func UpdateLDAPRealm(realm LDAPRealm) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&realm.Realm).Error; err != nil {
			logger.Error("Failed to update associated Realm for LDAP realm", "error", err)
			return err
		}

		if err := tx.Save(&realm).Error; err != nil {
			logger.Error("Failed to update LDAP realm", "error", err)
			return err
		}

		logger.Debug("LDAP realm updated successfully", "realmID", realm.ID)
		return nil
	})
}
