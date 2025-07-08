package api

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

func getUserIDFromContext(r *http.Request) uint {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userIDFloat, ok := claims[CLAIM_USER_ID].(float64)
	if !ok {
		logger.With("claims", claims[CLAIM_USER_ID]).Error("User ID claim not found or not a float64")
		panic("User ID claim not found or not a float64")
	}

	return uint(userIDFloat)
}
