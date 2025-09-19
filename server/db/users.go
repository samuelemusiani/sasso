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
var ErrPasswordRequired = errors.New("password is required for local realm")

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Password []byte
	Email    string   `gorm:"uniqueIndex;not null"`
	Realm    string   `gorm:"default:'local'"`
	Role     UserRole `gorm:"type:varchar(20);not null;default:'user';check:role IN ('admin','user','mantainer')"`

	MaxCores uint `gorm:"not null;default:2"`
	MaxRAM   uint `gorm:"not null;default:2048"`
	MaxDisk  uint `gorm:"not null;default:4"`
	MaxNets  uint `gorm:"not null;default:1"`

	VPNConfig *string `gorm:"default:null"`

	VMs     []VM     `gorm:"foreignKey:UserID"`
	Nets    []Net    `gorm:"foreignKey:UserID"`
	SSHKeys []SSHKey `gorm:"foreignKey:UserID"`
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

	if u.Realm == "local" && len(u.Password) == 0 {
		return ErrPasswordRequired
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

func GetUserByID(id uint) (User, error) {
	var user User
	result := db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		} else {
			logger.With("error", result.Error).Error("Failed to retrieve user by ID")
			return User{}, result.Error
		}
	}

	return user, nil
}

func GetAllUsers() ([]User, error) {
	var users []User
	result := db.Find(&users)
	if result.Error != nil {
		logger.With("error", result.Error).Error("Failed to retrieve all users")
		return nil, result.Error
	}

	return users, nil
}

func CreateUser(user *User) error {
	result := db.Create(user)
	if result.Error != nil {
		logger.With("error", result.Error).Error("Failed to create user")
		return result.Error
	}
	return nil
}

func UpdateUser(user *User) error {
	result := db.Save(user)
	if result.Error != nil {
		logger.With("error", result.Error).Error("Failed to update user")
		return result.Error
	}
	return nil
}

func UpdateUserLimits(userID uint, maxCores uint, maxRAM uint, maxDisk uint, maxNets uint) error {
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to find user by ID")
		return err
	}

	result := db.Model(&user).
		Select("max_cores", "max_ram", "max_disk", "max_nets").
		Updates(&User{
			MaxCores: maxCores,
			MaxRAM:   maxRAM,
			MaxDisk:  maxDisk,
			MaxNets:  maxNets,
		})
	if result.Error != nil {
		logger.With("error", result.Error).Error("Failed to update user limits")
		return result.Error
	}
	return nil
}

func UpdateVPNConfig(config string, userID uint) error {
	if err := db.Model(&User{}).Where("id = ?", userID).Update("vpn_config", config).Error; err != nil {
		logger.With("userID", userID, "error", err).Error("Failed to update VPN config")
		return err
	}

	return nil
}
