package db

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleAdmin      UserRole = "admin"
	RoleUser       UserRole = "user"
	RoleMaintainer UserRole = "mantainer"
)

var ErrInvalidUserRole = errors.New("invalid user role")

type User struct {
	gorm.Model
	Username string   `gorm:"uniqueIndex;not null"`
	Password []byte   `gorm:"not null"`
	Email    string   `gorm:"uniqueIndex;not null"`
	Realm    string   `gorm:"default:'local'"`
	Role     UserRole `gorm:"type:enum('admin','user','mantainer');not null"`
}

func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleUser, RoleMaintainer:
		return true
	default:
		return false
	}
}

func (u *User) BeforeSave(tx *gorm.DB) error {
	if !u.Role.IsValid() {
		return ErrInvalidUserRole
	}
	return nil
}

func initUsers() error {
	err := db.AutoMigrate(&User{})
	if err != nil {
		logger.With("error", err).Error("Failed to migrate Users table")
		return err
	}

	var adminUser User
	result := db.First(&adminUser, "username = ?", "admin")
	if result.Error == nil {
		logger.Debug("Admin user already exists")
		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Some other error occurred
		logger.With("error", result.Error).Error("Failed to check for admin user")
		return result.Error
	}

	adminUser = User{
		Username: "admin",
		Email:    "admin@local",
		Realm:    "local",
		Role:     RoleAdmin,
	}

	passwd := rand.Text()
	adminUser.Password, err = bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		logger.With("error", err).Error("Failed to hash password")
		return err
	}

	if err := db.Create(&adminUser).Error; err != nil {
		logger.With("error", err).Error("Failed to create admin user")
		return err
	}

	logger.With("password", passwd).Info("Admin user created successfully")
	return nil
}

func GetUserByUsername(username string) (User, error) {
	var user User
	result := db.First(&user, "username = ?", username)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		} else {
			logger.With("error", result.Error).Error("Failed to retrieve user by username")
			return User{}, result.Error
		}
	}

	return user, nil
}
