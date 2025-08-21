package api

import (
	"encoding/json"
	"io"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"time"

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
	Realm    string      `json:"realm"`
	Role     db.UserRole `json:"role"`
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

	w.Header().Set("Content-Type", "application/json")
	returnUser := returnUser{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Realm:    user.Realm,
		Role:     user.Role,
	}

	err = json.NewEncoder(w).Encode(returnUser)
	if err != nil {
		logger.Error("failed to encode user to JSON", "error", err)
		http.Error(w, "Failed to encode user to JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.GetAllUsers()
	if err != nil {
		logger.Error("failed to get all users", "error", err)
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	returnUsers := make([]returnUser, len(users))
	for i, user := range users {
		returnUsers[i] = returnUser{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Realm:    user.Realm,
			Role:     user.Role,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(returnUsers); err != nil {
		logger.Error("failed to encode users to JSON", "error", err)
		http.Error(w, "Failed to encode users to JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
