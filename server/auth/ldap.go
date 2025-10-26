package auth

import (
	"fmt"

	"samuelemusiani/sasso/server/db"

	"github.com/go-ldap/ldap/v3"
)

type ldapAuthenticator struct {
	ID              uint
	URL             string
	UserBaseDN      string
	GroupBaseDN     string
	BindDN          string
	Password        string
	MaintainerGroup string
	AdminGroup      string
}

func (a *ldapAuthenticator) Login(username, password string) (*db.User, error) {
	l, err := ldap.DialURL(a.URL)
	if err != nil {
		logger.Error("Failed to connect to LDAP server", "url", a.URL, "error", err)
		return nil, err
	}
	defer l.Close()

	err = l.Bind(a.BindDN, a.Password)
	if err != nil {
		logger.Error("Failed to bind to LDAP server", "bindDN", a.BindDN, "error", err)
		return nil, err
	}

	searchRequest := ldap.NewSearchRequest(
		a.UserBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", username),
		[]string{"dn", "mail"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		logger.Error("Failed to search for user in LDAP", "baseDN", a.UserBaseDN, "username", username, "error", err)
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

	err = l.Bind(a.BindDN, a.Password)
	if err != nil {
		logger.Error("Failed to bind to LDAP server", "bindDN", a.BindDN, "error", err)
		return nil, err
	}

	var role db.UserRole = db.RoleUser

	if a.AdminGroup != "" {
		searchRequestGroup := ldap.NewSearchRequest(
			a.GroupBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf("(&(objectClass=groupOfNames)(cn=%s)(member=%s))", a.AdminGroup, userDN),
			[]string{"cn"},
			nil,
		)
		src, err := l.Search(searchRequestGroup)
		if err != nil {
			logger.Error("Failed to search for group in LDAP", "baseDN", a.UserBaseDN, "group", a.AdminGroup, "error", err)
			return nil, err
		}

		if len(src.Entries) == 1 {
			role = db.RoleAdmin
		} else {
			logger.Debug("Ldap search for admin group returned no entries", "err", err)
		}
	}
	if a.MaintainerGroup != "" && role == db.RoleUser {
		searchRequestGroup := ldap.NewSearchRequest(
			a.GroupBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf("(&(objectClass=groupOfNames)(cn=%s)(member=%s))", a.MaintainerGroup, userDN),
			[]string{"cn"},
			nil,
		)
		src, err := l.Search(searchRequestGroup)
		if err != nil {
			logger.Error("Failed to search for group in LDAP", "baseDN", a.UserBaseDN, "group", a.MaintainerGroup, "error", err)
			return nil, err
		}

		if len(src.Entries) == 1 {
			role = db.RoleMaintainer
		} else {
			logger.Debug("Ldap search for maintainer group returned no entries", "err", err)
		}
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		if err == db.ErrNotFound {
			logger.Info("User not found in local DB, creating new user", "username", username)

			newUser := db.User{
				Username: username,
				Password: nil, // Password is not stored for external users
				Email:    email,
				Role:     role,
				RealmID:  a.ID,
			}

			err = db.CreateUser(&newUser)
			if err != nil {
				logger.Error("Failed to create new user in local DB", "username", username, "error", err)
				return nil, err
			}
			return &newUser, nil
		}
		logger.Error("Failed to get user by username from local DB", "username", username, "error", err)
		return nil, err
	}

	// Update email if it has changed
	if user.Email != email || user.Role != role {
		user.Email = email
		user.Role = role
		err = db.UpdateUser(&user)
		if err != nil {
			// Log the error but continue, as the user is authenticated
			logger.Error("Failed to update user email", "error", err, "username", username, "role", role)
		}
	}

	return &user, nil
}

func (a *ldapAuthenticator) LoadConfigFromDB(realmID uint) error {
	ldapRealm, err := db.GetLDAPRealmByID(realmID)
	if err != nil {
		logger.Error("Failed to get LDAP realm by ID", "realmID", realmID, "error", err)
		return err
	}

	a.ID = ldapRealm.ID
	a.URL = ldapRealm.URL
	a.UserBaseDN = ldapRealm.UserBaseDN
	a.GroupBaseDN = ldapRealm.GroupBaseDN
	a.BindDN = ldapRealm.BindDN
	a.Password = ldapRealm.Password
	a.MaintainerGroup = ldapRealm.MaintainerGroup
	a.AdminGroup = ldapRealm.AdminGroup

	return nil
}
