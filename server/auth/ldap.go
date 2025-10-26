package auth

import (
	"errors"
	"net/url"
	"strings"

	"samuelemusiani/sasso/server/db"

	"github.com/go-ldap/ldap/v3"
)

type ldapAuthenticator struct {
	ID          uint
	URL         string
	UserBaseDN  string
	GroupBaseDN string
	BindDN      string
	Password    string

	LoginFilter      string
	MaintainerFilter string
	AdminFilter      string

	MailAttribute string
}

func (a *ldapAuthenticator) Login(username, password string) (*db.User, error) {
	l, err := ldap.DialURL(a.URL)
	if err != nil {
		logger.Error("Failed to connect to LDAP server", "url", a.URL, "error", err)
		return nil, err
	}
	defer l.Close()

	a.BindDN = ldap.EscapeDN(a.BindDN)

	err = l.Bind(a.BindDN, a.Password)
	if err != nil {
		logger.Error("Failed to bind to LDAP server", "bindDN", a.BindDN, "error", err)
		return nil, err
	}

	// LoginFilter is in the form (&(objectClass=person)(uid={{username}}))
	lgQueryFilter := strings.Replace(a.LoginFilter, "{{username}}", username, -1)
	lgQueryFilter = ldap.EscapeFilter(lgQueryFilter)

	logger.Debug("LDAP login filter", "filter", lgQueryFilter)

	searchRequest := ldap.NewSearchRequest(
		a.UserBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		lgQueryFilter, []string{"dn", a.MailAttribute},
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

	userDN := ldap.EscapeDN(sr.Entries[0].DN)
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

	var role db.UserRole = db.RoleUser

	if a.AdminFilter != "" {

		// AdminFilter is in the form (&(objectClass=groupOfNames)(cn=sass_admin)(member={{user_dn}}))

		adminGroupFilter := strings.Replace(a.AdminFilter, "{{user_dn}}", userDN, -1)
		adminGroupFilter = ldap.EscapeFilter(adminGroupFilter)

		logger.Debug("LDAP admin group filter", "filter", adminGroupFilter)

		searchRequestGroup := ldap.NewSearchRequest(
			a.GroupBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			adminGroupFilter,
			[]string{"cn"},
			nil,
		)
		src, err := l.Search(searchRequestGroup)
		if err != nil {
			logger.Error("Failed to search for admin group in LDAP", "baseDN", a.UserBaseDN, "error", err)
			return nil, err
		}

		if len(src.Entries) == 1 {
			role = db.RoleAdmin
		} else {
			logger.Debug("Ldap search for admin group returned no entries", "err", err)
		}
	}
	if a.MaintainerFilter != "" && role == db.RoleUser {
		// MaintainerFilter is in the form (&(objectClass=groupOfNames)(cn=sass_maintainer)(member={{user_dn}}))
		maintainerGroupFilter := strings.Replace(a.MaintainerFilter, "{{user_dn}}", userDN, -1)
		maintainerGroupFilter = ldap.EscapeFilter(maintainerGroupFilter)

		searchRequestGroup := ldap.NewSearchRequest(
			a.GroupBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			maintainerGroupFilter,
			[]string{"cn"},
			nil,
		)
		src, err := l.Search(searchRequestGroup)
		if err != nil {
			logger.Error("Failed to search for maintainer group in LDAP", "baseDN", a.UserBaseDN, "error", err)
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
	a.LoginFilter = ldapRealm.LoginFilter
	a.MaintainerFilter = ldapRealm.MaintainerFilter
	a.AdminFilter = ldapRealm.AdminFilter
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

func VerifyLDAPFilters(login, maintainer, admin string) error {
	if login == "" {
		return errors.Join(ErrInvalidConfig, errors.New("login filter cannot be empty"))
	}

	if strings.Index(login, "{{username}}") == -1 {
		return errors.Join(ErrInvalidConfig, errors.New("login filter must contain {{username}} placeholder"))
	}

	if admin != "" && strings.Index(admin, "{{user_dn}}") == -1 {
		return errors.Join(ErrInvalidConfig, errors.New("admin filter must contain {{user_dn}} placeholder"))
	}

	if maintainer != "" && strings.Index(maintainer, "{{user_dn}}") == -1 {
		return errors.Join(ErrInvalidConfig, errors.New("maintainer filter must contain {{user_dn}} placeholder"))
	}
	return nil
}

func VerifyLDAPAttribute(mailAttr string) error {
	if mailAttr == "" {
		return errors.Join(ErrInvalidConfig, errors.New("mail attribute cannot be empty"))
	}
	return nil
}
