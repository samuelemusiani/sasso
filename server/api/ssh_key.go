package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/notify"
	"strconv"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/ssh"
)

func getSSHKeys(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	keys, err := db.GetSSHKeysByUserID(userID)
	if err != nil {
		logger.Error("Failed to get SSH keys", "userID", userID, "error", err)
		http.Error(w, "Failed to get SSH keys", http.StatusInternalServerError)
		return
	}

	resp := make([]newSSHKeyResponse, len(keys))
	for i := range keys {
		resp[i] = newSSHKeyResponse{
			ID:   keys[i].ID,
			Name: keys[i].Name,
			Key:  keys[i].Key,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode SSH keys to JSON", "error", err)
		http.Error(w, "Failed to encode SSH keys to JSON", http.StatusInternalServerError)
		return
	}
}

type newSSHKeyRequest struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type newSSHKeyResponse struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Key    string `json:"key"`
	Global bool   `json:"global"`
}

func addSSHKey(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var req newSSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Key == "" {
		http.Error(w, "Name and Key are required", http.StatusBadRequest)
		return
	}

	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.Key))
	if err != nil {
		http.Error(w, "Invalid SSH key format", http.StatusBadRequest)
		return
	}

	key, err := db.CreateSSHKey(req.Name, req.Key, userID)
	if err != nil {
		logger.Error("Failed to add new SSH key", "userID", userID, "error", err)
		http.Error(w, "Failed to add new SSH key", http.StatusInternalServerError)
		return
	}

	resp := newSSHKeyResponse{
		ID:   key.ID,
		Name: key.Name,
		Key:  key.Key,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode new SSH key to JSON", "error", err)
		http.Error(w, "Failed to encode new SSH key to JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func deleteSSHKey(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	skeyID := chi.URLParam(r, "id")

	keyID, err := strconv.ParseUint(skeyID, 10, 32)
	if err != nil {
		logger.Error("Invalid SSH key ID format", "userID", userID, "keyID", skeyID, "error", err)
		http.Error(w, "Invalid SSH key ID format", http.StatusBadRequest)
		return
	}

	if err := db.DeleteSSHKey(uint(keyID), userID); err != nil {
		logger.Error("Failed to delete SSH key", "userID", userID, "keyID", keyID, "error", err)
		http.Error(w, "Failed to delete SSH key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getGlobalSSHKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := db.GetGlobalSSHKeys()
	if err != nil {
		logger.Error("Failed to get global SSH keys", "error", err)
		http.Error(w, "Failed to get global SSH keys", http.StatusInternalServerError)
		return
	}

	resp := make([]newSSHKeyResponse, len(keys))
	for i := range keys {
		resp[i] = newSSHKeyResponse{
			ID:     keys[i].ID,
			Name:   keys[i].Name,
			Key:    keys[i].Key,
			Global: keys[i].Global,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode global SSH keys to JSON", "error", err)
		http.Error(w, "Failed to encode global SSH keys to JSON", http.StatusInternalServerError)
		return
	}
}

func addGlobalSSHKey(w http.ResponseWriter, r *http.Request) {
	var req newSSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	key, err := db.CreateGlobalSSHKey(req.Name, req.Key)
	if err != nil {
		logger.Error("Failed to add new global SSH key", "error", err)
		http.Error(w, "Failed to add new global SSH key", http.StatusInternalServerError)
		return
	}

	resp := newSSHKeyResponse{
		ID:     key.ID,
		Name:   key.Name,
		Key:    key.Key,
		Global: key.Global,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode new global SSH key to JSON", "error", err)
		http.Error(w, "Failed to encode new global SSH key to JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	err = notify.SendGlobalSSHKeysChangeNotification()
	if err != nil {
		logger.Error("Failed to send global SSH keys change notification", "error", err)
	}
}

func deleteGlobalSSHKey(w http.ResponseWriter, r *http.Request) {
	skeyID := chi.URLParam(r, "id")

	keyID, err := strconv.ParseUint(skeyID, 10, 32)
	if err != nil {
		logger.Error("Invalid SSH key ID format", "keyID", skeyID, "error", err)
		http.Error(w, "Invalid SSH key ID format", http.StatusBadRequest)
		return
	}

	if err := db.DeleteGlobalSSHKey(uint(keyID)); err != nil {
		logger.Error("Failed to delete global SSH key", "keyID", keyID, "error", err)
		http.Error(w, "Failed to delete global SSH key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	err = notify.SendGlobalSSHKeysChangeNotification()
	if err != nil {
		logger.Error("Failed to send global SSH keys change notification", "error", err)
	}
}
