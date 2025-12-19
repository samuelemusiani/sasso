package auth

import (
	"errors"
	"net/url"
	"slices"
	"strings"

	"samuelemusiani/sasso/server/db"

	"github.com/go-ldap/ldap/v3"
)

type ldapAuthenticator struct {
	ID         uint
	URL        string
	UserBaseDN string
	BindDN     string
	Password   string

	LoginFilter       string
	MaintainerGroupDN string
	AdminGroupDN      string

	MailAttribute string
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

	// LoginFilter is in the form (&(objectClass=person)(uid={{username}}))
	lgQueryFilter := strings.ReplaceAll(a.LoginFilter, "{{username}}", username)

	logger.Debug("LDAP login filter", "filter", lgQueryFilter)

	searchRequest := ldap.NewSearchRequest(
		a.UserBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		lgQueryFilter, []string{"dn", a.MailAttribute, "memberOf"},
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

	email := sr.Entries[0].GetAttributeValue(a.MailAttribute)

	err = l.Bind(a.BindDN, a.Password)
	if err != nil {
		logger.Error("Failed to bind to LDAP server", "bindDN", a.BindDN, "error", err)
		return nil, err
	}

	role := db.RoleUser

	if a.AdminGroupDN != "" {
		if slices.Contains(sr.Entries[0].GetAttributeValues("memberOf"), a.AdminGroupDN) {
			role = db.RoleAdmin
		}
	}
	if a.MaintainerGroupDN != "" && role == db.RoleUser {
		if slices.Contains(sr.Entries[0].GetAttributeValues("memberOf"), a.MaintainerGroupDN) {
			role = db.RoleMaintainer
		}
	}

	user, err := db.GetUserByUsernameAndRealmID(username, a.ID)
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
	a.BindDN = ldapRealm.BindDN
	a.Password = ldapRealm.Password
	a.LoginFilter = ldapRealm.LoginFilter
	a.MaintainerGroupDN = ldapRealm.MaintainerGroupDN
	a.AdminGroupDN = ldapRealm.AdminGroupDN
	a.MailAttribute = ldapRealm.MailAttribute

	return nil
}
func VerifyLDAPURL(lurl string) error {
	if lurl == "" {
		return errors.Join(ErrInvalidConfig, errors.New("LDAP URL cannot be empty"))
	}
	ldapURL, err := url.Parse(lurl)
	if err != nil || (ldapURL.Scheme != "ldap" && ldapURL.Scheme != "ldaps") {
		return errors.Join(ErrInvalidConfig, errors.New("invalid LDAP URL"))
	}
	return nil
}

func VerifyLDAPLoginFilter(login string) error {
	if login == "" {
		return errors.Join(ErrInvalidConfig, errors.New("login filter cannot be empty"))
	}

	if !strings.Contains(login, "{{username}}") {
		return errors.Join(ErrInvalidConfig, errors.New("login filter must contain {{username}} placeholder"))
	}
	return nil
}

func VerifyLDAPAttribute(mailAttr string) error {
	if mailAttr == "" {
		return errors.Join(ErrInvalidConfig, errors.New("mail attribute cannot be empty"))
	}
	return nil
}
