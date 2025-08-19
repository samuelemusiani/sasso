package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/db"
)

type returnRealm struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
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
