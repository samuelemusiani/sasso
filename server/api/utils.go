package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/go-ldap/ldap/v3"
	"golang.org/x/crypto/bcrypt"
)

func getUserIDFromContext(r *http.Request) (uint, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		logger.Error("Failed to get claims from context", "error", err)
		return 0, err
	}
	// All JSON numbers are decoded into float64 by default
	userID, ok := claims[CLAIM_USER_ID].(float64)
	if !ok {
		logger.Error("User ID claim not found or not a float64", "claims", claims[CLAIM_USER_ID])
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

func (a *LocalAuthenticator) LoadConfigFromDB(realmID uint) error {
	// Local authentication does not require any specific configuration from the database
	return nil
}

type LDAPAuthenticator struct {
	ID              uint
	URL             string
	UserBaseDN      string
	GroupBaseDN     string
	BindDN          string
	Password        string
	MaintainerGroup string
	AdminGroup      string
}

func (a *LDAPAuthenticator) Login(username, password string) (*db.User, error) {
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

func (a *LDAPAuthenticator) LoadConfigFromDB(realmID uint) error {
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

func authenticator(username, password string, realm uint) (*db.User, error) {
	dbRealm, err := db.GetRealmByID(realm)
	if err != nil {
		logger.Error("Failed to get realm by ID", "realmID", realm, "error", err)
		return nil, err
	}

	var l Authenticator
	switch dbRealm.Type {
	case db.LocalRealmType:
		logger.Debug("Using local authentication for realm", "realmID", realm)
		l = &LocalAuthenticator{}
	case db.LDAPRealmType:
		logger.Debug("Using LDAP authentication for realm", "realmID", realm)
		l = &LDAPAuthenticator{}
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

func validateVMOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromContext(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			svmID := chi.URLParam(r, "vmid")
			vmID, err := strconv.ParseUint(svmID, 10, 64)
			if err != nil {
				http.Error(w, "invalid vm id", http.StatusBadRequest)
				return
			}

			vm, err := proxmox.GetVMByID(vmID, userID)
			if err != nil {
				http.Error(w, "vm not found", http.StatusNotFound)
				return
			}

			var role string
			if vm.OwnerType == "User" && vm.OwnerID != userID {
				http.Error(w, "vm does not belong to the user", http.StatusForbidden)
				return
			}
			if vm.OwnerType == "Group" {
				role, err = db.GetUserRoleInGroup(userID, vm.OwnerID)
				if err != nil {
					if errors.Is(err, db.ErrNotFound) {
						http.Error(w, "vm does not belong to the user", http.StatusForbidden)
						return
					}
					logger.Error("failed to get user role in group", "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				// TODO: check role permissions if needed?
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "vm_id", vm)
			ctx = context.WithValue(ctx, "group_user_role", role)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(hfn)
	}
}

func mustGetVMFromContext(r *http.Request) *proxmox.VM {
	vm, ok := r.Context().Value("vm_id").(*proxmox.VM)
	if !ok {
		panic("getVMFromContext: vm_id not found in context")
	}
	return vm
}

func mustGetUserRoleInGroupFromContext(r *http.Request) string {
	role, ok := r.Context().Value("group_user_role").(string)
	if !ok {
		panic("mustGetUserRoleInGroupFromContext: group_user_role not found in context")
	}
	return role
}

func validateInterfaceOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromContext(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			sifaceID := chi.URLParam(r, "ifaceid")
			ifaceID, err := strconv.ParseUint(sifaceID, 10, 32)
			if err != nil {
				http.Error(w, "invalid interface id", http.StatusBadRequest)
				return
			}

			iface, err := db.GetInterfaceByID(uint(ifaceID))
			if err != nil {
				http.Error(w, "interface not found", http.StatusBadRequest)
				return
			}

			n, err := db.GetNetByID(iface.VNetID)
			if err != nil {
				http.Error(w, "vnet not found", http.StatusBadRequest)
				return
			}

			// TODO: group vnets
			if n.OwnerType == "User" && n.OwnerID != userID {
				http.Error(w, "vnet does not belong to the user", http.StatusForbidden)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "interface_id", iface)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(hfn)
	}
}

func mustGetInterfaceFromContext(r *http.Request) *db.Interface {
	iface, ok := r.Context().Value("interface_id").(*db.Interface)
	if !ok {
		panic("getInterfaceFromContext: interface_id not found in context")
	}
	return iface
}

func validateGroupOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromContext(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			sgroupID := chi.URLParam(r, "groupid")
			groupID, err := strconv.ParseUint(sgroupID, 10, 32)
			if err != nil {
				http.Error(w, "invalid group id", http.StatusBadRequest)
				return
			}

			group, err := db.GetGroupByID(uint(groupID))
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					http.Error(w, "group not found", http.StatusBadRequest)
					return
				}
				logger.Error("failed to get group by id", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			userRole, err := db.GetUserRoleInGroup(userID, group.ID)
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					http.Error(w, "group not found", http.StatusNotFound)
					return
				}
				logger.Error("failed to get user role in group", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "group", group)
			ctx = context.WithValue(ctx, "group_user_role", userRole)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(hfn)
	}
}

func mustGetGroupFromContext(r *http.Request) *db.Group {
	group, ok := r.Context().Value("group").(*db.Group)
	if !ok {
		panic("getGroupFromContext: group not found in context")
	}
	return group
}
