package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type returnGroup struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	Role string `json:"role,omitempty"`

	Members []returnGroupMember `json:"members,omitempty"`
}

type returnGroupMember struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func listUserGroups(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	groups, err := db.GetGroupsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve groups", http.StatusInternalServerError)
		return
	}
	returnGroups := make([]returnGroup, 0, len(groups))
	for _, group := range groups {
		returnGroups = append(returnGroups, returnGroup{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Role:        group.Role,
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
		if err == db.ErrNotFound {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getGroup(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)
	returnGroup := returnGroup{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
	}

	members, err := db.GetGroupMembers(group.ID)
	if err != nil {
		http.Error(w, "Failed to retrieve group members", http.StatusInternalServerError)
		return
	}
	returnGroup.Members = make([]returnGroupMember, 0, len(members))
	for _, member := range members {
		returnGroup.Members = append(returnGroup.Members, returnGroupMember{
			UserID:   member.UserID,
			Username: member.Username,
			Role:     member.Role,
		})
	}

	if err := json.NewEncoder(w).Encode(returnGroup); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

type returnGroupInvitation struct {
	ID               uint   `json:"id"`
	GroupID          uint   `json:"group_id"`
	UserID           uint   `json:"user_id"`
	Role             string `json:"role"`
	State            string `json:"state"`
	Username         string `json:"username"`
	GroupName        string `json:"group_name"`
	GroupDescription string `json:"group_description"`
}

func listGroupInvitations(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	invitations, err := db.GetGroupsWithInvitationByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve invitations", http.StatusInternalServerError)
		return
	}

	returnInvitations := make([]returnGroupInvitation, 0, len(invitations))
	for _, inv := range invitations {
		returnInvitations = append(returnInvitations, returnGroupInvitation{
			ID:               inv.ID,
			GroupID:          inv.GroupID,
			UserID:           inv.UserID,
			Role:             inv.Role,
			State:            inv.State,
			Username:         inv.Username,
			GroupName:        inv.GroupName,
			GroupDescription: inv.GroupDescription,
		})
	}

	if err := json.NewEncoder(w).Encode(returnInvitations); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

type manageInvitationsRequest struct {
	Action string `json:"action"` // "accept" or "decline"
}

func manageInvitation(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	sInvitationID := chi.URLParam(r, "inviteid")
	invitationID, err := strconv.ParseUint(sInvitationID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
		return
	}

	var req manageInvitationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Action != "accept" && req.Action != "decline" {
		http.Error(w, "Invalid invitation ID or action", http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "accept":
		if err := db.AcceptGroupInvitation(uint(invitationID), userID); err != nil {
			http.Error(w, "Failed to accept invitation", http.StatusInternalServerError)
			return
		}
	case "decline":
		if err := db.DeclineGroupInvitation(uint(invitationID), userID); err != nil {
			http.Error(w, "Failed to decline invitation", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

func getGroupPendingInvitations(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)

	invitations, err := db.GetPendingGroupInvitationsByGroupID(group.ID)
	if err != nil {
		http.Error(w, "Failed to retrieve invitations", http.StatusInternalServerError)
		return
	}

	returnInvitations := make([]returnGroupInvitation, 0, len(invitations))
	for _, inv := range invitations {
		returnInvitations = append(returnInvitations, returnGroupInvitation{
			ID:               inv.ID,
			GroupID:          inv.GroupID,
			UserID:           inv.UserID,
			Role:             inv.Role,
			State:            inv.State,
			Username:         inv.Username,
			GroupName:        inv.GroupName,
			GroupDescription: inv.GroupDescription,
		})
	}

	if err := json.NewEncoder(w).Encode(returnInvitations); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

type requestInviteUser struct {
	Username string `json:"username"`
	Role     string `json:"role"` // "member" or "admin"
}

func inviteUserToGroup(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)

	var req requestInviteUser
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || (req.Role != "member" && req.Role != "admin") {
		http.Error(w, "Invalid username or role", http.StatusBadRequest)
		return
	}

	u, err := db.GetUserByUsername(req.Username)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve user", http.StatusInternalServerError)
		return
	}

	belongs, err := db.DoesUserBelongToGroup(u.ID, group.ID)
	if err != nil {
		http.Error(w, "Failed to check user group membership", http.StatusInternalServerError)
		return
	}
	if belongs {
		http.Error(w, "User already in group", http.StatusConflict)
		return
	}

	if err := db.InviteUserToGroup(u.ID, group.ID, req.Role); err != nil {
		if err == db.ErrAlreadyExists {
			http.Error(w, "User already invited", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to invite user", http.StatusInternalServerError)
		return
	}
}

func revokeGroupInvitation(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)
	sInviteID := chi.URLParam(r, "inviteid")
	inviteID, err := strconv.ParseUint(sInviteID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
		return
	}

	if getUserRoleInGroupFromContext(r) != "owner" {
		http.Error(w, "Only group owners can revoke invitations", http.StatusForbidden)
		return
	}

	if err := db.RevokeGroupInvitationToUser(uint(inviteID), group.ID); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Invitation not found or already processed", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to revoke invitation", http.StatusInternalServerError)
		return
	}
}

func listGroupMembers(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)

	members, err := db.GetGroupMembers(group.ID)
	if err != nil {
		http.Error(w, "Failed to retrieve group members", http.StatusInternalServerError)
		return
	}

	returnMembers := make([]returnGroupMember, 0, len(members))
	for _, member := range members {
		returnMembers = append(returnMembers, returnGroupMember{
			UserID:   member.UserID,
			Username: member.Username,
			Role:     member.Role,
		})
	}

	if err := json.NewEncoder(w).Encode(returnMembers); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func leaveGroup(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)
	userID := mustGetUserIDFromContext(r)
	user_role := getUserRoleInGroupFromContext(r)

	if user_role == "owner" {
		http.Error(w, "Group owners cannot leave the group. Transfer ownership or delete the group.", http.StatusForbidden)
		return
	}

	if err := db.RemoveUserFromGroup(userID, group.ID); err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "You are not a member of this group", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to leave group", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func removeUserFromGroup(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)
	user_role := getUserRoleInGroupFromContext(r)

	if user_role != "owner" {
		http.Error(w, "Only group owners can remove members", http.StatusForbidden)
		return
	}

	sUserID := chi.URLParam(r, "userid")
	userID, err := strconv.ParseUint(sUserID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := db.RemoveUserFromGroup(uint(userID), group.ID); err != nil {
		http.Error(w, "Failed to remove user from group", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getMyGroupMembership(w http.ResponseWriter, r *http.Request) {
	group := getGroupFromContext(r)
	userID := mustGetUserIDFromContext(r)

	role, err := db.GetUserRoleInGroup(userID, group.ID)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "You are not a member of this group", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve user role in group", http.StatusInternalServerError)
		return
	}

	returnMember := returnGroupMember{
		UserID:   userID,
		Role:     role,
		Username: "", // Username is not needed here
	}

	if err := json.NewEncoder(w).Encode(returnMember); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
