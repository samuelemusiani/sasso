package db

import (
	"crypto/rand"
	"errors"
	"fmt"

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

	Username string `gorm:"not null;uniqueIndex:idx_username_realm"`
	Password []byte
	Email    string   `gorm:"not null;uniqueIndex:idx_email_realm"`
	RealmID  uint     `gorm:"not null;uniqueIndex:idx_username_realm;uniqueIndex:idx_email_realm"`
	Role     UserRole `gorm:"type:varchar(20);not null;default:'user';check:role IN ('admin','user','maintainer')"`

	MaxCores uint `gorm:"not null;default:2"`
	MaxRAM   uint `gorm:"not null;default:2048"`
	MaxDisk  uint `gorm:"not null;default:4"`
	MaxNets  uint `gorm:"not null;default:1"`

	VMs            []VM            `gorm:"polymorphic:Owner;polymorphicValue:User"`
	Nets           []Net           `gorm:"polymorphic:Owner;polymorphicValue:User"`
	SSHKeys        []SSHKey        `gorm:"foreignKey:UserID"`
	PortForwards   []PortForward   `gorm:"polymorphic:Owner;polymorphicValue:User"`
	BackupRequests []BackupRequest `gorm:"polymorphic:Owner;polymorphicValue:User"`
	// Notifications  []Notification  `gorm:"foreignKey:UserID"`
	// We can't have notifications here because we set UserID to 0 for global notifications
	TelegramBots   []TelegramBot   `gorm:"foreignKey:UserID"`
	WireguardPeers []WireguardPeer `gorm:"foreignKey:UserID"`

	Groups        []Group         `gorm:"many2many:user_groups;"`
	GroupResource []GroupResource `gorm:"foreignKey:UserID"`

	Settings Setting `gorm:"foreignKey:UserID"`
}

func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleUser, RoleMaintainer:
		return true
	default:
		return false
	}
}

func (u *User) BeforeSave(_ *gorm.DB) error {
	if !u.Role.IsValid() {
		return ErrInvalidUserRole
	}

	return nil
}

func getLocalRealmIDTransaction(tx *gorm.DB) (uint, error) {
	var realmID uint

	err := tx.Select("id").Where("name = ?", "Local").First(&Realm{}).Scan(&realmID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrNotFound
		}

		return 0, fmt.Errorf("failed to get local realm ID: %w", err)
	}

	return realmID, nil
}

func initUsers() error {
	err := db.AutoMigrate(&User{})
	if err != nil {
		return fmt.Errorf("failed to migrate users table: %w", err)
	}

	var adminUser User

	result := db.First(&adminUser, "username = ?", "admin")
	if result.Error == nil {
		logger.Debug("Admin user already exists")

		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Some other error occurred
		return fmt.Errorf("failed to check for admin user: %w", result.Error)
	}

	localRealmID, err := getLocalRealmIDTransaction(db)
	if err != nil {
		return fmt.Errorf("failed to get local realm ID: %w", err)
	}

	adminUser = User{
		Username: "admin",
		Email:    "admin@local",
		RealmID:  localRealmID,
		Role:     RoleAdmin,
	}

	passwd := rand.Text()

	adminUser.Password, err = bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := CreateUser(&adminUser); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	s := `===============================================================
Admin user created successfully. Password: %s
===============================================================
`

	_, err = fmt.Printf(s, passwd)
	if err != nil {
		return fmt.Errorf("failed to print admin password: %w", err)
	}

	return nil
}

func UpdateAdminPassword(password string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		adminID, err := getAdminIDTransaction(tx)
		if err != nil {
			return fmt.Errorf("failed to get adminID: %w", err)
		}

		var admin User

		err = tx.First(&admin, adminID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}

			return fmt.Errorf("failed to get admin user: %w", err)
		}

		admin.Password, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		err = tx.Save(&admin).Error
		if err != nil {
			return err
		}

		return nil
	})
}

func GetUserByUsernameAndRealmID(username string, realmID uint) (User, error) {
	var user User

	result := db.Where(&User{Username: username, RealmID: realmID}).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		}

		return User{}, fmt.Errorf("failed to retrieve user by username: %w", result.Error)
	}

	return user, nil
}

func GetUserByID(id uint) (User, error) {
	var user User

	result := db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		}

		return User{}, fmt.Errorf("failed to retrieve user by ID: %w", result.Error)
	}

	return user, nil
}

func GetAllUsers() ([]User, error) {
	var users []User

	result := db.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve all users: %w", result.Error)
	}

	return users, nil
}

func CreateUser(user *User) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Create(user)
		if result.Error != nil {
			return fmt.Errorf("failed to create user: %w", result.Error)
		}

		return createDefaultSettingsForUserTransaction(tx, user.ID)
	})

	return err
}

func UpdateUser(user *User) error {
	result := db.Save(user)
	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}

	return nil
}

func UpdateUserLimits(userID uint, maxCores uint, maxRAM uint, maxDisk uint, maxNets uint) error {
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}

		return fmt.Errorf("failed to find user by ID: %w", err)
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
		return fmt.Errorf("failed to update user limits: %w", result.Error)
	}

	return nil
}

func GetAllUserEmails() ([]string, error) {
	var emails []string
	if err := db.Model(&User{}).Where("id != ?", 1).Pluck("email", &emails).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve all user emails: %w", err)
	}

	return emails, nil
}

func getAdminIDTransaction(tx *gorm.DB) (uint, error) {
	var adminID uint

	err := tx.Raw(`
			SELECT users.id
			FROM users
			JOIN realms ON users.realm_id = realms.id
			WHERE realms.name = 'Local' AND users.username = 'admin'
		`).Scan(&adminID).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get admin ID: %w", err)
	}

	return adminID, nil
}

func GetLocalAdmin() (*User, error) {
	var admin User

	err := db.Joins("JOIN realms ON users.realm_id = realms.id").
		Where("realms.name = ? AND users.username = ?", "Local", "admin").
		First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to get local admin: %w", err)
	}

	return &admin, nil
}
