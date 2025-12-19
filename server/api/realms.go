package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"samuelemusiani/sasso/server/auth"
	"samuelemusiani/sasso/server/db"
)

type Realm struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type LDAPRealm struct {
	Realm

	URL        string `json:"url"`
	UserBaseDN string `json:"user_base_dn"`
	BindDN     string `json:"bind_dn"`
	Password   string `json:"password,omitempty"`

	LoginFilter       string `json:"login_filter"`
	MaintainerGroupDN string `json:"maintainer_group_dn"`
	AdminGroupDN      string `json:"admin_group_dn"`

	MailAttribute string `json:"mail_attribute"`
}

type UpdateLDAPRealm struct {
	Realm

	URL        *string `json:"url"`
	UserBaseDN *string `json:"user_base_dn"`
	BindDN     *string `json:"bind_dn"`
	Password   *string `json:"password,omitempty"`

	LoginFilter       *string `json:"login_filter"`
	MaintainerGroupDN *string `json:"maintainer_group_dn"`
	AdminGroupDN      *string `json:"admin_group_dn"`

	MailAttribute *string `json:"mail_attribute"`
}

func listRealms(w http.ResponseWriter, r *http.Request) {
	realms, err := db.GetAllRealms()
	if err != nil {
		logger.Error("Failed to get realms", "error", err)
		http.Error(w, "Failed to get realms", http.StatusInternalServerError)

		return
	}

	returnedRealms := make([]Realm, len(realms))
	for i, realm := range realms {
		returnedRealms[i] = Realm{
			ID:          realm.ID,
			Name:        realm.Name,
			Description: realm.Description,
			Type:        realm.Type,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(returnedRealms); err != nil {
		logger.Error("Failed to encode realms to JSON", "error", err)
		http.Error(w, "Failed to encode realms to JSON", http.StatusInternalServerError)

		return
	}
}

func addRealm(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	var newRealm Realm
	if err := json.Unmarshal(body, &newRealm); err != nil {
		logger.Error("Failed to unmarshal request body into Realm", "error", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)

		return
	}

	if newRealm.Name == "" {
		logger.Error("Realm name cannot be empty")
		http.Error(w, "Realm name cannot be empty", http.StatusBadRequest)

		return
	}

	switch newRealm.Type {
	case "local":
		http.Error(w, "Local realm cannot be added via API", http.StatusBadRequest)
		return

	case "ldap":
		var ldapRealm LDAPRealm
		if err := json.Unmarshal(body, &ldapRealm); err != nil {
			logger.Error("Failed to unmarshal request body into LDAPRealm", "error", err)
			http.Error(w, "Invalid JSON format for LDAP realm", http.StatusBadRequest)

			return
		}

		if ldapRealm.URL == "" || ldapRealm.UserBaseDN == "" ||
			ldapRealm.BindDN == "" || ldapRealm.Password == "" ||
			ldapRealm.LoginFilter == "" {
			logger.Error("Missing required fields for LDAP realm")
			http.Error(w, "Missing required fields for LDAP realm", http.StatusBadRequest)

			return
		}

		err = auth.VerifyLDAPURL(ldapRealm.URL)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidConfig) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				logger.Error("Failed to verify LDAP URL", "error", err)
				http.Error(w, "Failed to verify LDAP URL", http.StatusInternalServerError)
			}

			return
		}

		err = auth.VerifyLDAPLoginFilter(ldapRealm.LoginFilter)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidConfig) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				logger.Error("Failed to verify LDAP filters", "error", err)
				http.Error(w, "Failed to verify LDAP filters", http.StatusInternalServerError)
			}

			return
		}

		err = auth.VerifyLDAPAttribute(ldapRealm.MailAttribute)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidConfig) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				logger.Error("Failed to verify LDAP mail attribute", "error", err)
				http.Error(w, "Failed to verify LDAP mail attribute", http.StatusInternalServerError)
			}

			return
		}

		logger.Info("Adding new LDAP realm", "ldapRealm", ldapRealm)

		dbRealm := db.LDAPRealm{
			Realm: db.Realm{
				Name:        newRealm.Name,
				Description: newRealm.Description,
				Type:        newRealm.Type,
			},
			URL:               ldapRealm.URL,
			UserBaseDN:        ldapRealm.UserBaseDN,
			BindDN:            ldapRealm.BindDN,
			Password:          ldapRealm.Password,
			LoginFilter:       ldapRealm.LoginFilter,
			MaintainerGroupDN: ldapRealm.MaintainerGroupDN,
			AdminGroupDN:      ldapRealm.AdminGroupDN,
			MailAttribute:     ldapRealm.MailAttribute,
		}
		if err := db.AddLDAPRealm(dbRealm); err != nil {
			logger.Error("Failed to add LDAP realm", "error", err)
			http.Error(w, "Failed to add LDAP realm", http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Unsupported realm type", http.StatusBadRequest)
		return
	}
}

func getRealm(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	srealmID := chi.URLParam(r, "id")

	realmID, err := strconv.ParseUint(srealmID, 10, 32)
	if err != nil {
		logger.Error("Invalid Realm ID format", "userID", userID, "srealmID", srealmID, "error", err)
		http.Error(w, "Invalid Realm ID format", http.StatusBadRequest)

		return
	}

	realm, err := db.GetRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.Error("Realm not found", "userID", userID, "realmID", realmID)
			http.Error(w, "Realm not found", http.StatusNotFound)

			return
		}

		logger.Error("Failed to get Realm by ID", "userID", userID, "realmID", realmID, "error", err)
		http.Error(w, "Failed to get Realm by ID", http.StatusInternalServerError)

		return
	}

	var returnedRealm any

	basicRealm := Realm{
		ID:          realm.ID,
		Name:        realm.Name,
		Description: realm.Description,
		Type:        realm.Type,
	}

	switch realm.Type {
	case "local":
		returnedRealm = basicRealm
	case "ldap":
		ldapRealm, err := db.GetLDAPRealmByID(uint(realmID))
		if err != nil {
			if err == db.ErrNotFound {
				logger.Error("LDAP Realm not found", "userID", userID, "realmID", realmID)
				http.Error(w, "LDAP Realm not found", http.StatusNotFound)

				return
			}

			logger.Error("Failed to get LDAP Realm by ID", "userID", userID, "realmID", realmID, "error", err)
			http.Error(w, "Failed to get LDAP Realm by ID", http.StatusInternalServerError)

			return
		}

		returnedRealm = LDAPRealm{
			Realm:             basicRealm,
			URL:               ldapRealm.URL,
			UserBaseDN:        ldapRealm.UserBaseDN,
			BindDN:            ldapRealm.BindDN,
			LoginFilter:       ldapRealm.LoginFilter,
			MaintainerGroupDN: ldapRealm.MaintainerGroupDN,
			AdminGroupDN:      ldapRealm.AdminGroupDN,
			MailAttribute:     ldapRealm.MailAttribute,
		}
	default:
		logger.Error("Unsupported realm type", "userID", userID, "realmID", realmID)
		http.Error(w, "Unsupported realm type", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(returnedRealm); err != nil {
		logger.Error("Failed to encode Realm to JSON", "error", err)
		http.Error(w, "Failed to encode Realm to JSON", http.StatusInternalServerError)

		return
	}
}

func deleteRealm(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	srealmID := chi.URLParam(r, "id")

	realmID, err := strconv.ParseUint(srealmID, 10, 32)
	if err != nil {
		logger.Error("Invalid Realm ID format", "userID", userID, "srealmID", srealmID, "error", err)
		http.Error(w, "Invalid Realm ID format", http.StatusBadRequest)

		return
	}

	realm, err := db.GetRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.Error("Realm not found", "userID", userID, "realmID", realmID)
			http.Error(w, "Realm not found", http.StatusNotFound)

			return
		}

		logger.Error("Failed to get Realm by ID", "userID", userID, "realmID", realmID, "error", err)
		http.Error(w, "Failed to get Realm by ID", http.StatusInternalServerError)

		return
	}

	if realm.Type == "local" {
		logger.Error("Cannot delete local realm via API", "userID", userID, "realmID", realmID)
		http.Error(w, "Cannot delete local realm via API", http.StatusBadRequest)

		return
	}

	err = db.DeleteRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.Error("Realm not found", "userID", userID, "realmID", realmID)
			http.Error(w, "Realm not found", http.StatusNotFound)

			return
		}

		logger.Error("Failed to delete Realm by ID", "userID", userID, "realmID", realmID, "error", err)
		http.Error(w, "Failed to delete Realm by ID", http.StatusInternalServerError)

		return
	}

	logger.Info("Realm deleted successfully", "userID", userID, "realmID", realmID)
	w.WriteHeader(http.StatusNoContent)
}

func updateRealm(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	srealmID := chi.URLParam(r, "id")

	realmID, err := strconv.ParseUint(srealmID, 10, 32)
	if err != nil {
		logger.Error("Invalid Realm ID format", "userID", userID, "srealmID", srealmID, "error", err)
		http.Error(w, "Invalid Realm ID format", http.StatusBadRequest)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	realm, err := db.GetRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.Error("Realm not found", "userID", userID, "realmID", realmID)
			http.Error(w, "Realm not found", http.StatusNotFound)

			return
		}

		logger.Error("Failed to get Realm by ID", "userID", userID, "realmID", realmID, "error", err)
		http.Error(w, "Failed to get Realm by ID", http.StatusInternalServerError)

		return
	}

	switch realm.Type {
	case "local":
		http.Error(w, "Local realm cannot be updated via API", http.StatusBadRequest)
		return
	case "ldap":
		ldapRealm, err := db.GetLDAPRealmByID(uint(realmID))
		if err != nil {
			if err == db.ErrNotFound {
				logger.Error("LDAP Realm not found", "userID", userID, "realmID", realmID)
				http.Error(w, "LDAP Realm not found", http.StatusNotFound)

				return
			}

			logger.Error("Failed to get LDAP Realm by ID", "userID", userID, "realmID", realmID, "error", err)
			http.Error(w, "Failed to get LDAP Realm by ID", http.StatusInternalServerError)

			return
		}

		var clientLdapRealm UpdateLDAPRealm

		err = json.Unmarshal(body, &clientLdapRealm)
		if err != nil {
			logger.Error("Failed to unmarshal request body into Realm", "error", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)

			return
		}

		if clientLdapRealm.Name != "" {
			ldapRealm.Name = clientLdapRealm.Name
		}

		if clientLdapRealm.Description != "" {
			ldapRealm.Description = clientLdapRealm.Description
		}

		if clientLdapRealm.URL != nil {
			ldapRealm.URL = *clientLdapRealm.URL
		}

		if clientLdapRealm.UserBaseDN != nil {
			ldapRealm.UserBaseDN = *clientLdapRealm.UserBaseDN
		}

		if clientLdapRealm.BindDN != nil {
			ldapRealm.BindDN = *clientLdapRealm.BindDN
		}

		if clientLdapRealm.Password != nil {
			ldapRealm.Password = *clientLdapRealm.Password
		}

		if clientLdapRealm.LoginFilter != nil {
			ldapRealm.LoginFilter = *clientLdapRealm.LoginFilter
		}

		if clientLdapRealm.MaintainerGroupDN != nil {
			ldapRealm.MaintainerGroupDN = *clientLdapRealm.MaintainerGroupDN
		}

		if clientLdapRealm.AdminGroupDN != nil {
			ldapRealm.AdminGroupDN = *clientLdapRealm.AdminGroupDN
		}

		if clientLdapRealm.MailAttribute != nil {
			ldapRealm.MailAttribute = *clientLdapRealm.MailAttribute
		}

		err = auth.VerifyLDAPURL(ldapRealm.URL)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidConfig) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				logger.Error("Failed to verify LDAP URL", "error", err)
				http.Error(w, "Failed to verify LDAP URL", http.StatusInternalServerError)
			}

			return
		}

		err = auth.VerifyLDAPLoginFilter(ldapRealm.LoginFilter)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidConfig) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				logger.Error("Failed to verify LDAP login filter", "error", err)
				http.Error(w, "Failed to verify LDAP login filter", http.StatusInternalServerError)
			}

			return
		}

		err = auth.VerifyLDAPAttribute(ldapRealm.MailAttribute)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidConfig) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				logger.Error("Failed to verify LDAP mail attribute", "error", err)
				http.Error(w, "Failed to verify LDAP mail attribute", http.StatusInternalServerError)
			}

			return
		}

		if err := db.UpdateLDAPRealm(*ldapRealm); err != nil {
			logger.Error("Failed to update LDAP Realm", "userID", userID, "realmID", realmID, "error", err)
			http.Error(w, "Failed to update LDAP Realm", http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		logger.Error("Unsupported realm type", "userID", userID, "realmID", realmID)
		http.Error(w, "Unsupported realm type", http.StatusInternalServerError)

		return
	}
}
