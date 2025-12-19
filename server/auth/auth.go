package auth

import (
	"errors"
	"log/slog"

	"samuelemusiani/sasso/server/db"
)

var (
	logger *slog.Logger = nil

	ErrUserNotFound     = errors.New("user not found")
	ErrPasswordMismatch = errors.New("password mismatch")
	ErrTooManyUsers     = errors.New("too many users found")

	ErrInvalidConfig = errors.New("invalid realm configuration")
)

func Init(l *slog.Logger) error {
	logger = l
	return nil
}

type Authenticator interface {
	// This function is used to authenticate a user based on their username, password
	// If the user exists in an external realm, a local user will be created.
	// If the user already exists in the database, it will be updated.
	Login(username, password string) (*db.User, error)
	LoadConfigFromDB(realmID uint) error
}

// Authenticate authenticates a user based on the provided username, password,
// and realm ID.
func Authenticate(username, password string, realm uint) (*db.User, error) {
	dbRealm, err := db.GetRealmByID(realm)
	if err != nil {
		logger.Error("Failed to get realm by ID", "realmID", realm, "error", err)
		return nil, err
	}

	var l Authenticator

	switch dbRealm.Type {
	case db.LocalRealmType:
		logger.Debug("Using local authentication for realm", "realmID", realm)

		l = &localAuthenticator{}
	case db.LDAPRealmType:
		logger.Debug("Using LDAP authentication for realm", "realmID", realm)

		l = &ldapAuthenticator{}
	default:
		logger.Error("Unsupported realm type for authentication", "realmType", dbRealm.Type)
		return nil, errors.New("unsupported realm type for authentication")
	}

	err = l.LoadConfigFromDB(realm)
	if err != nil {
		logger.Error("Failed to load realm configuration from database", "realmID", realm, "error", err)
		return nil, err
	}

	user, err := l.Login(username, password)
	if err != nil {
		logger.Error("Failed to authenticate user", "username", username, "realmID", realm, "error", err)
		return nil, err
	}

	return user, nil
}
