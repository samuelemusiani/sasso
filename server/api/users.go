package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

const CLAIM_USER_ID = "user_id"

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

	user, err := authenticator(loginReq.Username, loginReq.Password, loginReq.Realm)
	if err != nil {
		if err == ErrUserNotFound {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		} else if err == ErrPasswordMismatch {
			http.Error(w, "Password mismatch", http.StatusUnauthorized)
			return
		} else {
			logger.Error("failed to authenticate user", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	logger.Info("User authenticated successfully", "userID", user.ID)

	// Password matches, create JWT token
	claims := map[string]any{CLAIM_USER_ID: user.ID}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiryIn(claims, time.Hour*12) // Set token expiry to 24 hours

	_, tokenString, err := tokenAuth.Encode(claims)
	if err != nil {
		logger.Error("failed to create JWT token", "error", err)
		http.Error(w, "Failed to create JWT token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+tokenString)
	w.Write([]byte("Login successful!"))
}

func whoami(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		logger.Error("failed to get user ID from context", "error", err)
		w.Write([]byte("unauthenticated"))
		return
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		if err == db.ErrNotFound {
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

func listUsers(w http.ResponseWriter, r *http.Request) {
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

	var realmMap map[uint]string = make(map[uint]string)
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
	userID, err := strconv.ParseUint(suserID, 10, 64)
	if err != nil {
		logger.Error("failed to parse user ID", "error", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := db.GetUserByID(uint(userID))
	if err != nil {
		if err == db.ErrNotFound {
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
}

func getUserResources(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	var userResources returnUserResources
	var err error

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

	userResources.AllocatedCores, userResources.AllocatedRAM, userResources.AllocatedDisk, err = db.GetVMResourcesByUserID(userID)
	if err != nil {
		logger.Error("failed to get VM resources by user ID", "error", err)
		http.Error(w, "Failed to get VM resources", http.StatusInternalServerError)
		return
	}
	userResources.AllocatedNets, err = db.CountNetsByUserID(userID)

	userResources.ActiveVMsCores, userResources.ActiveVMsRAM, userResources.ActiveVMsDisk, err = db.GetResorcesActiveVMsByUserID(userID)
	if err != nil {
		logger.Error("failed to get active VM resources by user ID", "error", err)
		http.Error(w, "Failed to get active VM resources", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(userResources); err != nil {
		logger.Error("failed to encode resources to JSON", "error", err)
		http.Error(w, "Failed to encode resources to JSON", http.StatusInternalServerError)
		return
	}
}
