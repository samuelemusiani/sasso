package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"samuelemusiani/sasso/server/auth"
	"samuelemusiani/sasso/server/db"
)

const ClaimUserID = "user_id"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Realm    uint   `json:"realm"`
}

type returnUser struct {
	ID       uint        `json:"id"`
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Realm    string      `json:"realm,omitempty"`
	Role     db.UserRole `json:"role"`
	MaxCores uint        `json:"max_cores,omitempty"`
	MaxRAM   uint        `json:"max_ram,omitempty"`
	MaxDisk  uint        `json:"max_disk,omitempty"`
	MaxNets  uint        `json:"max_nets,omitempty"`
}

func login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("failed to read body", "error", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)

		return
	}

	var loginReq loginRequest
	if err := json.Unmarshal(body, &loginReq); err != nil {
		logger.Error("failed to unmarshal login request", "error", err)
		http.Error(w, "Failed to unmarshal login request", http.StatusBadRequest)

		return
	}

	user, err := auth.Authenticate(loginReq.Username, loginReq.Password, loginReq.Realm)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			http.Error(w, "User not found", http.StatusUnauthorized)

			return
		case errors.Is(err, auth.ErrPasswordMismatch):
			http.Error(w, "Password mismatch", http.StatusUnauthorized)

			return
		default:
			logger.Error("failed to authenticate user", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)

			return
		}
	}

	logger.Info("User authenticated successfully", "userID", user.ID)

	// Password matches, create JWT token
	claims := map[string]any{ClaimUserID: user.ID}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiryIn(claims, time.Hour*12) // Set token expiry to 24 hours

	_, tokenString, err := tokenAuth.Encode(claims)
	if err != nil {
		logger.Error("failed to create JWT token", "error", err)
		http.Error(w, "Failed to create JWT token", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Authorization", "Bearer "+tokenString)

	_, err = w.Write([]byte("Login successful!"))
	if err != nil {
		logger.Error("failed to write login response", "error", err)
		http.Error(w, "Failed to write login response", http.StatusInternalServerError)

		return
	}
}

func whoami(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		logger.Error("failed to get user ID from context", "error", err)

		_, err = w.Write([]byte("unauthenticated"))
		if err != nil {
			logger.Error("failed to write unauthenticated response", "error", err)
			http.Error(w, "Failed to write response", http.StatusInternalServerError)

			return
		}

		return
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)

			return
		}

		logger.Error("failed to get user by ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	realm, err := db.GetRealmByID(user.RealmID)
	if err != nil {
		logger.Error("failed to get realm by ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	returnUser := returnUser{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Realm:    realm.Name,
		Role:     user.Role,
	}

	err = json.NewEncoder(w).Encode(returnUser)
	if err != nil {
		logger.Error("failed to encode user to JSON", "error", err)
		http.Error(w, "Failed to encode user to JSON", http.StatusInternalServerError)

		return
	}
}

func internalListUsers(w http.ResponseWriter, _ *http.Request) {
	users, err := db.GetAllUsers()
	if err != nil {
		logger.Error("failed to get all users", "error", err)
		http.Error(w, "Failed to get users", http.StatusInternalServerError)

		return
	}

	realms, err := db.GetAllRealms()
	if err != nil {
		logger.Error("failed to get all realms", "error", err)
		http.Error(w, "Failed to get realms", http.StatusInternalServerError)

		return
	}

	realmMap := make(map[uint]string)
	for _, realm := range realms {
		realmMap[realm.ID] = realm.Name
	}

	returnUsers := make([]returnUser, len(users))
	for i, user := range users {
		realm, ok := realmMap[user.RealmID]
		if !ok {
			slog.Error("realm not found for user", "userID", user.ID, "realmID", user.RealmID)

			realm = "unknown"
		}

		returnUsers[i] = returnUser{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Realm:    realm,
			Role:     user.Role,
			MaxCores: user.MaxCores,
			MaxRAM:   user.MaxRAM,
			MaxDisk:  user.MaxDisk,
			MaxNets:  user.MaxNets,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(returnUsers); err != nil {
		logger.Error("failed to encode users to JSON", "error", err)
		http.Error(w, "Failed to encode users to JSON", http.StatusInternalServerError)

		return
	}
}

func getUser(w http.ResponseWriter, r *http.Request) {
	suserID := chi.URLParam(r, "id")

	userID, err := strconv.ParseUint(suserID, 10, 32)
	if err != nil {
		logger.Error("failed to parse user ID", "error", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)

		return
	}

	user, err := db.GetUserByID(uint(userID))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)

			return
		}

		logger.Error("failed to get user by ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	realm, err := db.GetRealmByID(user.RealmID)
	if err != nil {
		logger.Error("failed to get realm by ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	returnUser := returnUser{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Realm:    realm.Name,
		Role:     user.Role,
		MaxCores: user.MaxCores,
		MaxRAM:   user.MaxRAM,
		MaxDisk:  user.MaxDisk,
		MaxNets:  user.MaxNets,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(returnUser); err != nil {
		logger.Error("failed to encode user to JSON", "error", err)
		http.Error(w, "Failed to encode user to JSON", http.StatusInternalServerError)

		return
	}
}

type updateUserLimitsRequest struct {
	UserID   uint `json:"user_id"`
	MaxCores uint `json:"max_cores"`
	MaxRAM   uint `json:"max_ram"`
	MaxDisk  uint `json:"max_disk"`
	MaxNets  uint `json:"max_nets"`
}

func updateUserLimits(w http.ResponseWriter, r *http.Request) {
	var req updateUserLimitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	if err := db.UpdateUserLimits(req.UserID, req.MaxCores, req.MaxRAM, req.MaxDisk, req.MaxNets); err != nil {
		http.Error(w, "Failed to update user limits", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type returnUserResources struct {
	MaxCores       uint `json:"max_cores"`
	MaxRAM         uint `json:"max_ram"`
	MaxDisk        uint `json:"max_disk"`
	MaxNets        uint `json:"max_nets"`
	AllocatedCores uint `json:"allocated_cores"`
	AllocatedRAM   uint `json:"allocated_ram"`
	AllocatedDisk  uint `json:"allocated_disk"`
	AllocatedNets  uint `json:"allocated_nets"`
	ActiveVMsCores uint `json:"active_vms_cores"`
	ActiveVMsRAM   uint `json:"active_vms_ram"`
	ActiveVMsDisk  uint `json:"active_vms_disk"`

	GroupMaxCores uint `json:"group_max_cores"`
	GroupMaxRAM   uint `json:"group_max_ram"`
	GroupMaxDisk  uint `json:"group_max_disk"`
	GroupMaxNets  uint `json:"group_max_nets"`
}

func getUserResources(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var (
		userResources returnUserResources
		err           error
	)

	user, err := db.GetUserByID(userID)
	if err != nil {
		logger.Error("failed to get user by ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	userResources.MaxCores = user.MaxCores
	userResources.MaxRAM = user.MaxRAM
	userResources.MaxDisk = user.MaxDisk
	userResources.MaxNets = user.MaxNets

	groupRes, err := db.GetUserGroupResourcesByUserID(userID)
	if err != nil {
		logger.Error("failed to get group resources by user ID", "error", err)
		http.Error(w, "Failed to get group resources", http.StatusInternalServerError)

		return
	}

	userResources.MaxCores += groupRes.Cores
	userResources.MaxRAM += groupRes.RAM
	userResources.MaxDisk += groupRes.Disk
	userResources.MaxNets += groupRes.Nets

	userResources.GroupMaxCores += groupRes.Cores
	userResources.GroupMaxRAM += groupRes.RAM
	userResources.GroupMaxDisk += groupRes.Disk
	userResources.GroupMaxNets += groupRes.Nets

	allocatedResources, err := db.GetVMResourcesByUserID(userID)
	if err != nil {
		logger.Error("failed to get VM resources by user ID", "error", err)
		http.Error(w, "Failed to get VM resources", http.StatusInternalServerError)

		return
	}

	userResources.AllocatedCores = allocatedResources.Cores
	userResources.AllocatedRAM = allocatedResources.RAM
	userResources.AllocatedDisk = allocatedResources.Disk

	userResources.AllocatedNets, err = db.CountNetsByUserID(userID)
	if err != nil {
		logger.Error("failed to count networks by user ID", "error", err)
		http.Error(w, "Failed to count networks", http.StatusInternalServerError)

		return
	}

	activeResources, err := db.GetResourcesActiveVMsByUserID(userID)
	if err != nil {
		logger.Error("failed to get active VM resources by user ID", "error", err)
		http.Error(w, "Failed to get active VM resources", http.StatusInternalServerError)

		return
	}

	userResources.ActiveVMsCores = activeResources.Cores
	userResources.ActiveVMsRAM = activeResources.RAM
	userResources.ActiveVMsDisk = activeResources.Disk

	if err := json.NewEncoder(w).Encode(userResources); err != nil {
		logger.Error("failed to encode resources to JSON", "error", err)
		http.Error(w, "Failed to encode resources to JSON", http.StatusInternalServerError)

		return
	}
}

type returnUserSettings struct {
	MailPortForwardNotification          bool `json:"mail_port_forward_notification"`
	MailVMStatusUpdateNotification       bool `json:"mail_vm_status_update_notification"`
	MailGlobalSSHKeysChangeNotification  bool `json:"mail_global_ssh_keys_change_notification"`
	MailVMExpirationNotification         bool `json:"mail_vm_expiration_notification"`
	MailVMEliminatedNotification         bool `json:"mail_vm_eliminated_notification"`
	MailVMStoppedNotification            bool `json:"mail_vm_stopped_notification"`
	MailSSHKeysChangedOnVMNotification   bool `json:"mail_ssh_keys_changed_on_vm_notification"`
	MailUserInvitationNotification       bool `json:"mail_user_invitation_notification"`
	MailUserRemovalFromGroupNotification bool `json:"mail_user_removal_from_group_notification"`
	MailLifetimeOfVMExpiredNotification  bool `json:"mail_lifetime_of_vm_expired_notification"`

	TelegramPortForwardNotification          bool `json:"telegram_port_forward_notification"`
	TelegramVMStatusUpdateNotification       bool `json:"telegram_vm_status_update_notification"`
	TelegramGlobalSSHKeysChangeNotification  bool `json:"telegram_global_ssh_keys_change_notification"`
	TelegramVMExpirationNotification         bool `json:"telegram_vm_expiration_notification"`
	TelegramVMEliminatedNotification         bool `json:"telegram_vm_eliminated_notification"`
	TelegramVMStoppedNotification            bool `json:"telegram_vm_stopped_notification"`
	TelegramSSHKeysChangedOnVMNotification   bool `json:"telegram_ssh_keys_changed_on_vm_notification"`
	TelegramUserInvitationNotification       bool `json:"telegram_user_invitation_notification"`
	TelegramUserRemovalFromGroupNotification bool `json:"telegram_user_removal_from_group_notification"`
	TelegramLifetimeOfVMExpiredNotification  bool `json:"telegram_lifetime_of_vm_expired_notification"`
}

func getUserSettings(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	settings, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("failed to get user settings", "error", err)
		http.Error(w, "Failed to get user settings", http.StatusInternalServerError)

		return
	}

	returnSettings := returnUserSettings{
		MailPortForwardNotification:          settings.MailPortForwardNotification,
		MailVMStatusUpdateNotification:       settings.MailVMStatusUpdateNotification,
		MailGlobalSSHKeysChangeNotification:  settings.MailGlobalSSHKeysChangeNotification,
		MailVMExpirationNotification:         settings.MailVMExpirationNotification,
		MailVMEliminatedNotification:         settings.MailVMEliminatedNotification,
		MailVMStoppedNotification:            settings.MailVMStoppedNotification,
		MailSSHKeysChangedOnVMNotification:   settings.MailSSHKeysChangedOnVMNotification,
		MailUserInvitationNotification:       settings.MailUserInvitationNotification,
		MailUserRemovalFromGroupNotification: settings.MailUserRemovalFromGroupNotification,
		MailLifetimeOfVMExpiredNotification:  settings.MailLifetimeOfVMExpiredNotification,

		TelegramPortForwardNotification:          settings.TelegramPortForwardNotification,
		TelegramVMStatusUpdateNotification:       settings.TelegramVMStatusUpdateNotification,
		TelegramGlobalSSHKeysChangeNotification:  settings.TelegramGlobalSSHKeysChangeNotification,
		TelegramVMExpirationNotification:         settings.TelegramVMExpirationNotification,
		TelegramVMEliminatedNotification:         settings.TelegramVMEliminatedNotification,
		TelegramVMStoppedNotification:            settings.TelegramVMStoppedNotification,
		TelegramSSHKeysChangedOnVMNotification:   settings.TelegramSSHKeysChangedOnVMNotification,
		TelegramUserInvitationNotification:       settings.TelegramUserInvitationNotification,
		TelegramUserRemovalFromGroupNotification: settings.TelegramUserRemovalFromGroupNotification,
		TelegramLifetimeOfVMExpiredNotification:  settings.TelegramLifetimeOfVMExpiredNotification,
	}

	if err := json.NewEncoder(w).Encode(returnSettings); err != nil {
		logger.Error("failed to encode user settings to JSON", "error", err)
		http.Error(w, "Failed to encode user settings to JSON", http.StatusInternalServerError)

		return
	}
}

func updateUserSettings(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var req returnUserSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("failed to decode user settings from JSON", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("failed to get user settings", "error", err)
		http.Error(w, "Failed to get user settings", http.StatusInternalServerError)

		return
	}

	s.MailPortForwardNotification = req.MailPortForwardNotification
	s.MailVMStatusUpdateNotification = req.MailVMStatusUpdateNotification
	s.MailGlobalSSHKeysChangeNotification = req.MailGlobalSSHKeysChangeNotification
	s.MailVMExpirationNotification = req.MailVMExpirationNotification
	s.MailVMEliminatedNotification = req.MailVMEliminatedNotification
	s.MailVMStoppedNotification = req.MailVMStoppedNotification
	s.MailSSHKeysChangedOnVMNotification = req.MailSSHKeysChangedOnVMNotification
	s.MailUserInvitationNotification = req.MailUserInvitationNotification
	s.MailUserRemovalFromGroupNotification = req.MailUserRemovalFromGroupNotification
	s.MailLifetimeOfVMExpiredNotification = req.MailUserRemovalFromGroupNotification

	s.TelegramPortForwardNotification = req.TelegramPortForwardNotification
	s.TelegramVMStatusUpdateNotification = req.TelegramVMStatusUpdateNotification
	s.TelegramGlobalSSHKeysChangeNotification = req.TelegramGlobalSSHKeysChangeNotification
	s.TelegramVMExpirationNotification = req.TelegramVMExpirationNotification
	s.TelegramVMEliminatedNotification = req.TelegramVMEliminatedNotification
	s.TelegramVMStoppedNotification = req.TelegramVMStoppedNotification
	s.TelegramSSHKeysChangedOnVMNotification = req.TelegramSSHKeysChangedOnVMNotification
	s.TelegramUserInvitationNotification = req.TelegramUserInvitationNotification
	s.TelegramUserRemovalFromGroupNotification = req.TelegramUserRemovalFromGroupNotification
	s.TelegramLifetimeOfVMExpiredNotification = req.TelegramUserRemovalFromGroupNotification

	if err := db.UpdateSettings(s); err != nil {
		logger.Error("failed to update user settings", "error", err)
		http.Error(w, "Failed to update user settings", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
