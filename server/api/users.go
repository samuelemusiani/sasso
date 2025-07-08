package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"samuelemusiani/sasso/server/db"

	"github.com/go-chi/jwtauth/v5"
	"golang.org/x/crypto/bcrypt"
)

const CLAIM_USER_ID = "user_id"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Realm    string `json:"realm"`
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

	user, err := db.GetUserByUsername(loginReq.Username)
	if err != nil {
		if err == db.ErrNotFound {
			logger.Info("user not found", "username", loginReq.Username)
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		} else {
			logger.Error("failed to get user by username", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	err = bcrypt.CompareHashAndPassword(user.Password, []byte(loginReq.Password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			logger.Info("password mismatch", "username", loginReq.Username)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		} else {
			logger.Error("failed to compare password", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Password matches, create JWT token
	_, tokenString, _ := tokenAuth.Encode(map[string]any{CLAIM_USER_ID: user.ID})

	w.Header().Set("Authorization", "Bearer "+tokenString)
	w.Write([]byte("Login successful!"))
}

func test(w http.ResponseWriter, r *http.Request) {
	// Placeholder for login logic
	_, claims, _ := jwtauth.FromContext(r.Context())
	w.Write(fmt.Appendf([]byte{}, "protected area. hi %v", claims[CLAIM_USER_ID]))
}
