package api

import (
	"errors"
	"net/http"

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
