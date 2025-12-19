package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"samuelemusiani/sasso/server/db"
)

type localAuthenticator struct {
	ID uint
}

func (a *localAuthenticator) Login(username, password string) (*db.User, error) {
	user, err := db.GetUserByUsernameAndRealmID(username, a.ID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
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
	realm, err := db.GetRealmByID(realmID)
	if err != nil {
		logger.Error("failed to get realm by ID", "realmID", realmID, "error", err)
		return err
	}

	a.ID = realm.ID

	return nil
}
