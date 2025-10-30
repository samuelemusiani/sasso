package auth

import (
	"errors"

	"samuelemusiani/sasso/server/db"

	"golang.org/x/crypto/bcrypt"
)

type localAuthenticator struct{}

func (a *localAuthenticator) Login(username, password string) (*db.User, error) {
	user, err := db.GetUserByUsername(username)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrUserNotFound
		} else {
			logger.Error("failed to get user by username", "error", err)
			return nil, err
		}
	}

	if user.RealmID != 1 {
		return nil, ErrUserNotFound
	}

	err = bcrypt.CompareHashAndPassword(user.Password, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			logger.Info("password mismatch", "username", username)
			return nil, ErrPasswordMismatch
		} else {
			logger.Error("failed to compare password", "error", err)
			return nil, err
		}
	}

	return &user, nil
}

func (a *localAuthenticator) LoadConfigFromDB(realmID uint) error {
	// Local authentication does not require any specific configuration from the database
	return nil
}
