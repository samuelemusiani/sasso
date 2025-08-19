package api

import (
	"errors"
	"net/http"
	"samuelemusiani/sasso/server/db"

	"github.com/go-chi/jwtauth/v5"
)

func getUserIDFromContext(r *http.Request) (uint, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		logger.With("error", err).Error("Failed to get claims from context")
		return 0, err
	}
	// All JSON numbers are decoded into float64 by default
	userID, ok := claims[CLAIM_USER_ID].(float64)
	if !ok {
		logger.With("claims", claims[CLAIM_USER_ID]).Error("User ID claim not found or not a float64")
		return 0, errors.New("user ID claim not found or not a float64")
	}

	return uint(userID), nil
}

func mustGetUserIDFromContext(r *http.Request) uint {
	id, err := getUserIDFromContext(r)
	if err != nil {
		panic("mustGetUserIDFromContext: " + err.Error())
	}
	return id
}

// AdminAuthenticator is an authentication middleware to enforce access from the
// Verifier middleware request context values. The Authenticator sends a 401 Unauthorized
// response for any unverified tokens and passes the good ones through.
func AdminAuthenticator(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, claims, err := jwtauth.FromContext(r.Context())

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if token == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID, ok := claims[CLAIM_USER_ID].(float64)
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			user, err := db.GetUserByID(uint(userID))
			if err != nil {
				if err == db.ErrNotFound {
					http.Error(w, "User not found", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Check if the user has admin role
			if user.Role != db.RoleAdmin {
				http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
				return
			}

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hfn)
	}
}
