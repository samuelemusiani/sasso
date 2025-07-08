package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

const CLAIM_USER_ID = "user_id"

func login(w http.ResponseWriter, r *http.Request) {
	_, tokenString, _ := tokenAuth.Encode(map[string]any{CLAIM_USER_ID: 123})

	w.Header().Set("Authorization", "Bearer "+tokenString)
	w.Write([]byte("Login endpoint not implemented yet"))
}

func test(w http.ResponseWriter, r *http.Request) {
	// Placeholder for login logic
	_, claims, _ := jwtauth.FromContext(r.Context())
	w.Write(fmt.Appendf([]byte{}, "protected area. hi %v", claims[CLAIM_USER_ID]))
}
