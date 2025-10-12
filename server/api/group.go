package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/db"
)

type returnGroup struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func listUserGroups(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	groups, err := db.GetGroupsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve groups", http.StatusInternalServerError)
		return
	}
	var returnGroups []returnGroup
	for _, group := range groups {
		returnGroups = append(returnGroups, returnGroup{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
		})
	}
	if err := json.NewEncoder(w).Encode(returnGroups); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func createGroup(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	if len(req.Name) > 64 {
		http.Error(w, "Group name too long", http.StatusBadRequest)
		return
	}

	if err := db.CreateGroup(req.Name, req.Description, userID); err != nil {
		http.Error(w, "Failed to create group", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func deleteGroup(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)
	user_role := getUserRoleInGroupFromContext(r)

	if user_role != "owner" {
		http.Error(w, "Only group owners can delete the group", http.StatusForbidden)
		return
	}

	if err := db.DeleteGroup(group.ID); err != nil {
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
