package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"samuelemusiani/sasso/server/db"
)

type returnRealm struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type returnLDAPRealm struct {
	returnRealm
	URL    string `json:"url"`
	BaseDN string `json:"base_dn"`
	BindDN string `json:"bind_dn"`
}

func listRealms(w http.ResponseWriter, r *http.Request) {
	realms, err := db.GetAllRealms()
	if err != nil {
		logger.With("error", err).Error("Failed to get realms")
		http.Error(w, "Failed to get realms", http.StatusInternalServerError)
		return
	}

	returnedRealms := make([]returnRealm, len(realms))
	for i, realm := range realms {
		returnedRealms[i] = returnRealm{
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
	w.WriteHeader(http.StatusOK)
}

func addRealm(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.With("error", err).Error("Failed to read request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var newRealm db.Realm
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
		var ldapRealm db.LDAPRealm
		if err := json.Unmarshal(body, &ldapRealm); err != nil {
			logger.With("error", err).Error("Failed to unmarshal request body into LDAPRealm")
			http.Error(w, "Invalid JSON format for LDAP realm", http.StatusBadRequest)
			return
		}

		if ldapRealm.URL == "" || ldapRealm.BaseDN == "" || ldapRealm.BindDN == "" || ldapRealm.Password == "" {
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

		logger.With("ldapRealm", ldapRealm).Info("Adding new LDAP realm")
		if err := db.AddLDAPRealm(ldapRealm); err != nil {
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
