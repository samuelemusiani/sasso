package api

import (
	"errors"
	"fmt"
	"net/http"
	"samuelemusiani/sasso/server/db"

	"github.com/go-chi/jwtauth/v5"
	"github.com/go-ldap/ldap/v3"
	"golang.org/x/crypto/bcrypt"
)

func getUserIDFromContext(r *http.Request) (uint, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		logger.With("error", err).Error("Failed to get claims from context")
		return 0, err
	}
	// All JSON numbers are decoded into float64 by default
	userID, ok := claims[CLAIM_USER_ID].(float64)
	if !ok {
		logger.With("claims", claims[CLAIM_USER_ID]).Error("User ID claim not found or not a float64")
		return 0, errors.New("user ID claim not found or not a float64")
	}

	return uint(userID), nil
}

func mustGetUserIDFromContext(r *http.Request) uint {
	id, err := getUserIDFromContext(r)
	if err != nil {
		panic("mustGetUserIDFromContext: " + err.Error())
	}
	return id
}

// AdminAuthenticator is an authentication middleware to enforce access from the
// Verifier middleware request context values. The Authenticator sends a 401 Unauthorized
// response for any unverified tokens and passes the good ones through.
func AdminAuthenticator(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, claims, err := jwtauth.FromContext(r.Context())

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if token == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID, ok := claims[CLAIM_USER_ID].(float64)
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			user, err := db.GetUserByID(uint(userID))
			if err != nil {
				if err == db.ErrNotFound {
					http.Error(w, "User not found", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Check if the user has admin role
			if user.Role != db.RoleAdmin {
				http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
				return
			}

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hfn)
	}
}

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrPasswordMismatch = errors.New("password mismatch")
	ErrTooManyUsers     = errors.New("too many users found with the same username")
)

type Authenticator interface {
	// This function is used to authenticate a user based on their username, password
	// If the user exists in an external realm, a local user will be created.
	// If the user already exists in the database, it will be updated.
	Login(username, password string) (*db.User, error)
	LoadConfigFromDB(realmID uint) error
}

type LocalAuthenticator struct{}

func (a *LocalAuthenticator) Login(username, password string) (*db.User, error) {
	user, err := db.GetUserByUsername(username)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrUserNotFound
		} else {
			logger.Error("failed to get user by username", "error", err)
			return nil, err
		}
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

func (a *LocalAuthenticator) LoadConfigFromDB(realmID uint) error {
	// Local authentication does not require any specific configuration from the database
	return nil
}

type LDAPAuthenticator struct {
	URL      string
	BaseDN   string
	BindDN   string
	Password string
}

func (a *LDAPAuthenticator) Login(username, password string) (*db.User, error) {
	l, err := ldap.DialURL(a.URL)
	if err != nil {
		logger.With("url", a.URL, "error", err).Error("Failed to connect to LDAP server")
		return nil, err
	}
	defer l.Close()

	err = l.Bind(a.BindDN, a.Password)
	if err != nil {
		logger.With("bindDN", a.BindDN, "error", err).Error("Failed to bind to LDAP server")
		return nil, err
	}

	searchRequest := ldap.NewSearchRequest(
		a.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", username),
		[]string{"dn", "mail"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		logger.With("baseDN", a.BaseDN, "username", username, "error", err).Error("Failed to search for user in LDAP")
		return nil, err
	}

	if len(sr.Entries) == 0 {
		return nil, ErrUserNotFound
	} else if len(sr.Entries) > 1 {
		return nil, ErrTooManyUsers
	}

	userDN := sr.Entries[0].DN
	err = l.Bind(userDN, password)
	if err != nil {
		return nil, ErrPasswordMismatch
	}

	email := sr.Entries[0].GetAttributeValue("mail")

	user, err := db.GetUserByUsername(username)
	if err != nil {
		if err == db.ErrNotFound {
			logger.With("username", username).Info("User not found in local DB, creating new user")

			newUser := db.User{
				Username: username,
				Password: nil, // Password is not stored for external users
				Email:    email,
				Role:     db.RoleUser,
				Realm:    db.LDAPRealmType,
			}

			err = db.CreateUser(&newUser)
			if err != nil {
				logger.With("username", username, "error", err).Error("Failed to create new user in local DB")
				return nil, err
			}
			return &newUser, nil
		}
		logger.With("username", username, "error", err).Error("Failed to get user by username from local DB")
		return nil, err
	}

	// Update email if it has changed
	if user.Email != email {
		user.Email = email
		err = db.UpdateUser(&user)
		if err != nil {
			// Log the error but continue, as the user is authenticated
			logger.With("error", err, "username", username).Error("Failed to update user email")
		}
	}

	return &user, nil
}

func (a *LDAPAuthenticator) LoadConfigFromDB(realmID uint) error {
	ldapRealm, err := db.GetLDAPRealmByID(realmID)
	if err != nil {
		logger.With("realmID", realmID, "error", err).Error("Failed to get LDAP realm by ID")
		return err
	}

	a.URL = ldapRealm.URL
	a.BaseDN = ldapRealm.BaseDN
	a.BindDN = ldapRealm.BindDN
	a.Password = ldapRealm.Password

	return nil
}

func authenticator(username, password string, realm uint) (*db.User, error) {
	dbRealm, err := db.GetRealmByID(realm)
	if err != nil {
		logger.With("realmID", realm, "error", err).Error("Failed to get realm by ID")
		return nil, err
	}

	var l Authenticator
	switch dbRealm.Type {
	case db.LocalRealmType:
		logger.With("realmID", realm).Debug("Using local authentication for realm")
		l = &LocalAuthenticator{}
	case db.LDAPRealmType:
		logger.With("realmID", realm).Debug("Using LDAP authentication for realm")
		l = &LDAPAuthenticator{}
	default:
		logger.With("realmType", dbRealm.Type).Error("Unsupported realm type for authentication")
		return nil, errors.New("unsupported realm type for authentication")
	}

	err = l.LoadConfigFromDB(realm)
	if err != nil {
		logger.With("realmID", realm, "error", err).Error("Failed to load realm configuration from database")
		return nil, err
	}

	user, err := l.Login(username, password)
	if err != nil {
		logger.With("username", username, "realmID", realm, "error", err).Error("Failed to authenticate user")
		return nil, err
	}

	return user, nil
}
