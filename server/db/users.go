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
	RoleMaintainer UserRole = "maintainer"
)

var ErrInvalidUserRole = errors.New("invalid user role")
var ErrPasswordRequired = errors.New("password is required for local realm")

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Password []byte
	Email    string   `gorm:"uniqueIndex;not null"`
	RealmID  uint     `gorm:"not null"`
	Role     UserRole `gorm:"type:varchar(20);not null;default:'user';check:role IN ('admin','user','maintainer')"`

	MaxCores uint `gorm:"not null;default:2"`
	MaxRAM   uint `gorm:"not null;default:2048"`
	MaxDisk  uint `gorm:"not null;default:4"`
	MaxNets  uint `gorm:"not null;default:1"`

	VPNConfig *string `gorm:"default:null"`

	VMs            []VM            `gorm:"polymorphic:Owner;polymorphicValue:User"`
	Nets           []Net           `gorm:"polymorphic:Owner;polymorphicValue:User"`
	SSHKeys        []SSHKey        `gorm:"foreignKey:UserID"`
	PortForwards   []PortForward   `gorm:"polymorphic:Owner;polymorphicValue:User"`
	BackupRequests []BackupRequest `gorm:"polymorphic:Owner;polymorphicValue:User"`
	// Notifications  []Notification  `gorm:"foreignKey:UserID"`
	// We can't have notifications here because we set UserID to 0 for global notifications
	TelegramBots []TelegramBot `gorm:"foreignKey:UserID"`

	Groups        []Group         `gorm:"many2many:user_groups;"`
	GroupResource []GroupResource `gorm:"foreignKey:UserID"`
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
		logger.Error("Failed to migrate Users table", "error", err)
		return err
	}

	var adminUser User
	result := db.First(&adminUser, "username = ?", "admin")
	if result.Error == nil {
		logger.Debug("Admin user already exists")
		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Some other error occurred
		logger.Error("Failed to check for admin user", "error", result.Error)
		return result.Error
	}

	adminUser = User{
		Username: "admin",
		Email:    "admin@local",
		RealmID:  1, // local realm: should have ID 1 as it's the first realm created
		Role:     RoleAdmin,
	}

	passwd := rand.Text()
	adminUser.Password, err = bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Failed to hash password", "error", err)
		return err
	}

	if err := db.Create(&adminUser).Error; err != nil {
		logger.Error("Failed to create admin user", "error", err)
		return err
	}

	logger.Debug("Admin user created successfully", "password", passwd)
	return nil
}

func GetUserByUsername(username string) (User, error) {
	var user User
	result := db.First(&user, "username = ?", username)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		} else {
			logger.Error("Failed to retrieve user by username", "error", result.Error)
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
			logger.Error("Failed to retrieve user by ID", "error", result.Error)
			return User{}, result.Error
		}
	}

	return user, nil
}

func GetAllUsers() ([]User, error) {
	var users []User
	result := db.Find(&users)
	if result.Error != nil {
		logger.Error("Failed to retrieve all users", "error", result.Error)
		return nil, result.Error
	}

	return users, nil
}

func CreateUser(user *User) error {
	result := db.Create(user)
	if result.Error != nil {
		logger.Error("Failed to create user", "error", result.Error)
		return result.Error
	}
	return nil
}

func UpdateUser(user *User) error {
	result := db.Save(user)
	if result.Error != nil {
		logger.Error("Failed to update user", "error", result.Error)
		return result.Error
	}
	return nil
}

func UpdateUserLimits(userID uint, maxCores uint, maxRAM uint, maxDisk uint, maxNets uint) error {
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		logger.Error("Failed to find user by ID", "userID", userID, "error", err)
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
		logger.Error("Failed to update user limits", "error", result.Error)
		return result.Error
	}
	return nil
}

func UpdateVPNConfig(config string, userID uint) error {
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		logger.Error("Failed to find user by ID", "userID", userID, "error", err)
		return err
	}

	if err := db.Model(&user).
		Select("vpn_config").
		Updates(&User{
			VPNConfig: &config,
		}).Error; err != nil {
		logger.Error("Failed to update VPN config", "userID", userID, "error", err)
		return err
	}

	return nil
}

func GetAllVPNConfigs() ([]User, error) {
	var users []User
	if err := db.Model(&User{}).Where("vpn_config IS NOT NULL").Select("id", "vpn_config").Find(&users).Error; err != nil {
		logger.Error("Failed to retrieve VPN configs", "error", err)
		return nil, err
	}
	return users, nil
}

func GetAllUserEmails() ([]string, error) {
	var emails []string
	if err := db.Model(&User{}).Where("id != ?", 1).Pluck("email", &emails).Error; err != nil {
		logger.Error("Failed to retrieve all user emails", "error", err)
		return nil, err
	}
	return emails, nil
}
