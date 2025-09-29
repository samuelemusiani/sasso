package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"samuelemusiani/sasso/server/db"

	"github.com/go-chi/chi/v5"
)

var groupRegex = regexp.MustCompile(`^\w*$`)

type returnLDAPRealm struct {
	Realm
	URL             string `json:"url"`
	UserBaseDN      string `json:"user_base_dn"`
	GroupBaseDN     string `json:"group_base_dn"`
	BindDN          string `json:"bind_dn"`
	MaintainerGroup string `json:"maintainer_group"`
	AdminGroup      string `json:"admin_group"`
}

type Realm struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type LDAPRealm struct {
	URL             string `json:"url"`
	UserBaseDN      string `json:"user_base_dn"`
	GroupBaseDN     string `json:"group_base_dn"`
	BindDN          string `json:"bind_dn"`
	Password        string `json:"password"`
	MaintainerGroup string `json:"maintainer_group"`
	AdminGroup      string `json:"admin_group"`
}

func listRealms(w http.ResponseWriter, r *http.Request) {
	realms, err := db.GetAllRealms()
	if err != nil {
		logger.With("error", err).Error("Failed to get realms")
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
		logger.With("error", err).Error("Failed to encode realms to JSON")
		http.Error(w, "Failed to encode realms to JSON", http.StatusInternalServerError)
		return
	}
}

func addRealm(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.With("error", err).Error("Failed to read request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var newRealm Realm
	if err := json.Unmarshal(body, &newRealm); err != nil {
		logger.With("error", err).Error("Failed to unmarshal request body into Realm")
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
			logger.With("error", err).Error("Failed to unmarshal request body into LDAPRealm")
			http.Error(w, "Invalid JSON format for LDAP realm", http.StatusBadRequest)
			return
		}

		if ldapRealm.URL == "" || ldapRealm.UserBaseDN == "" || ldapRealm.BindDN == "" || ldapRealm.Password == "" {
			logger.Error("Missing required fields for LDAP realm")
			http.Error(w, "Missing required fields for LDAP realm", http.StatusBadRequest)
			return
		}

		ldapURL, err := url.Parse(ldapRealm.URL)
		if err != nil || (ldapURL.Scheme != "ldap" && ldapURL.Scheme != "ldaps") {
			logger.With("ldapURL", ldapRealm.URL).Error("Invalid LDAP URL")
			http.Error(w, "Invalid LDAP URL", http.StatusBadRequest)
			return
		}

		if ldapRealm.MaintainerGroup != "" && !groupRegex.MatchString(ldapRealm.MaintainerGroup) {
			logger.With("maintainerGroup", ldapRealm.MaintainerGroup).Error("Invalid maintainer group format")
			http.Error(w, "Invalid maintainer group format", http.StatusBadRequest)
			return
		}

		if ldapRealm.AdminGroup != "" && !groupRegex.MatchString(ldapRealm.AdminGroup) {
			logger.With("adminGroup", ldapRealm.AdminGroup).Error("Invalid admin group format")
			http.Error(w, "Invalid admin group format", http.StatusBadRequest)
			return
		}

		if (ldapRealm.MaintainerGroup != "" || ldapRealm.AdminGroup != "") && ldapRealm.GroupBaseDN == "" {
			logger.With("groupBaseDN", ldapRealm.GroupBaseDN).Error("empty group base dn")
			http.Error(w, "Invalid group format", http.StatusBadRequest)
			return
		}

		logger.With("ldapRealm", ldapRealm).Info("Adding new LDAP realm")
		dbRealm := db.LDAPRealm{
			Realm: db.Realm{
				Name:        newRealm.Name,
				Description: newRealm.Description,
				Type:        newRealm.Type,
			},
			URL:             ldapRealm.URL,
			UserBaseDN:      ldapRealm.UserBaseDN,
			GroupBaseDN:     ldapRealm.GroupBaseDN,
			BindDN:          ldapRealm.BindDN,
			Password:        ldapRealm.Password,
			MaintainerGroup: ldapRealm.MaintainerGroup,
			AdminGroup:      ldapRealm.AdminGroup,
		}
		if err := db.AddLDAPRealm(dbRealm); err != nil {
			logger.With("error", err).Error("Failed to add LDAP realm")
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
	realmID, err := strconv.ParseUint(srealmID, 10, 64)
	if err != nil {
		logger.With("userID", userID, "srealmID", srealmID, "error", err).Error("Invalid Realm ID format")
		http.Error(w, "Invalid Realm ID format", http.StatusBadRequest)
		return
	}

	realm, err := db.GetRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.With("userID", userID, "realmID", realmID).Error("Realm not found")
			http.Error(w, "Realm not found", http.StatusNotFound)
			return
		}
		logger.With("userID", userID, "realmID", realmID, "error", err).Error("Failed to get Realm by ID")
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
				logger.With("userID", userID, "realmID", realmID).Error("LDAP Realm not found")
				http.Error(w, "LDAP Realm not found", http.StatusNotFound)
				return
			}
			logger.With("userID", userID, "realmID", realmID, "error", err).Error("Failed to get LDAP Realm by ID")
			http.Error(w, "Failed to get LDAP Realm by ID", http.StatusInternalServerError)
			return
		}

		returnedRealm = returnLDAPRealm{
			Realm:           basicRealm,
			URL:             ldapRealm.URL,
			UserBaseDN:      ldapRealm.UserBaseDN,
			GroupBaseDN:     ldapRealm.GroupBaseDN,
			BindDN:          ldapRealm.BindDN,
			MaintainerGroup: ldapRealm.MaintainerGroup,
			AdminGroup:      ldapRealm.AdminGroup,
		}
	default:
		logger.With("userID", userID, "realmID", realmID).Error("Unsupported realm type")
		http.Error(w, "Unsupported realm type", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(returnedRealm); err != nil {
		logger.With("error", err).Error("Failed to encode Realm to JSON")
		http.Error(w, "Failed to encode Realm to JSON", http.StatusInternalServerError)
		return
	}
}

func deleteRealm(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	srealmID := chi.URLParam(r, "id")
	realmID, err := strconv.ParseUint(srealmID, 10, 64)
	if err != nil {
		logger.With("userID", userID, "srealmID", srealmID, "error", err).Error("Invalid Realm ID format")
		http.Error(w, "Invalid Realm ID format", http.StatusBadRequest)
		return
	}

	realm, err := db.GetRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.With("userID", userID, "realmID", realmID).Error("Realm not found")
			http.Error(w, "Realm not found", http.StatusNotFound)
			return
		}
		logger.With("userID", userID, "realmID", realmID, "error", err).Error("Failed to get Realm by ID")
		http.Error(w, "Failed to get Realm by ID", http.StatusInternalServerError)
		return
	}

	if realm.Type == "local" {
		logger.With("userID", userID, "realmID", realmID).Error("Cannot delete local realm via API")
		http.Error(w, "Cannot delete local realm via API", http.StatusBadRequest)
		return
	}

	err = db.DeleteRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.With("userID", userID, "realmID", realmID).Error("Realm not found")
			http.Error(w, "Realm not found", http.StatusNotFound)
			return
		}
		logger.With("userID", userID, "realmID", realmID, "error", err).Error("Failed to delete Realm by ID")
		http.Error(w, "Failed to delete Realm by ID", http.StatusInternalServerError)
		return
	}
	logger.With("userID", userID, "realmID", realmID).Info("Realm deleted successfully")
	w.WriteHeader(http.StatusNoContent)
}

func updateRealm(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	srealmID := chi.URLParam(r, "id")
	realmID, err := strconv.ParseUint(srealmID, 10, 64)
	if err != nil {
		logger.With("userID", userID, "srealmID", srealmID, "error", err).Error("Invalid Realm ID format")
		http.Error(w, "Invalid Realm ID format", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.With("error", err).Error("Failed to read request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	realm, err := db.GetRealmByID(uint(realmID))
	if err != nil {
		if err == db.ErrNotFound {
			logger.With("userID", userID, "realmID", realmID).Error("Realm not found")
			http.Error(w, "Realm not found", http.StatusNotFound)
			return
		}
		logger.With("userID", userID, "realmID", realmID, "error", err).Error("Failed to get Realm by ID")
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
				logger.With("userID", userID, "realmID", realmID).Error("LDAP Realm not found")
				http.Error(w, "LDAP Realm not found", http.StatusNotFound)
				return
			}
			logger.With("userID", userID, "realmID", realmID, "error", err).Error("Failed to get LDAP Realm by ID")
			http.Error(w, "Failed to get LDAP Realm by ID", http.StatusInternalServerError)
			return
		}

		var clientLdapRealm db.LDAPRealm

		err = json.Unmarshal(body, &clientLdapRealm)
		if err != nil {
			logger.With("error", err).Error("Failed to unmarshal request body into Realm")
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if clientLdapRealm.Name != "" {
			ldapRealm.Name = clientLdapRealm.Name
		}
		if clientLdapRealm.Description != "" {
			ldapRealm.Description = clientLdapRealm.Description
		}
		if clientLdapRealm.URL != "" {
			ldapURL, err := url.Parse(clientLdapRealm.URL)
			if err != nil || (ldapURL.Scheme != "ldap" && ldapURL.Scheme != "ldaps") {
				logger.With("ldapURL", clientLdapRealm.URL).Error("Invalid LDAP URL")
				http.Error(w, "Invalid LDAP URL", http.StatusBadRequest)
				return
			}
			ldapRealm.URL = clientLdapRealm.URL
		}
		if clientLdapRealm.UserBaseDN != "" {
			ldapRealm.UserBaseDN = clientLdapRealm.UserBaseDN
		}
		if clientLdapRealm.GroupBaseDN != "" {
			ldapRealm.GroupBaseDN = clientLdapRealm.GroupBaseDN
		}
		if clientLdapRealm.BindDN != "" {
			ldapRealm.BindDN = clientLdapRealm.BindDN
		}
		if clientLdapRealm.Password != "" {
			ldapRealm.Password = clientLdapRealm.Password
		}
		if clientLdapRealm.MaintainerGroup != "" {
			ldapRealm.MaintainerGroup = clientLdapRealm.MaintainerGroup
		}
		if clientLdapRealm.AdminGroup != "" {
			ldapRealm.AdminGroup = clientLdapRealm.AdminGroup
		}

		if err := db.UpdateLDAPRealm(*ldapRealm); err != nil {
			logger.With("userID", userID, "realmID", realmID, "error", err).Error("Failed to update LDAP Realm")
			http.Error(w, "Failed to update LDAP Realm", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		logger.With("userID", userID, "realmID", realmID).Error("Unsupported realm type")
		http.Error(w, "Unsupported realm type", http.StatusInternalServerError)
		return
	}
}
