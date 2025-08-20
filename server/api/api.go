package api

import (
	"log/slog"
	"net/http"

	"samuelemusiani/sasso/server/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
)

var router *chi.Mux = nil
var logger *slog.Logger = nil

var tokenAuth *jwtauth.JWTAuth = nil

func Init(apiLogger *slog.Logger, key []byte) {
	// Logger
	logger = apiLogger

	// Router
	router = chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.CleanPath)

	apiRouter := chi.NewRouter()

	apiRouter.Use(middleware.Heartbeat("/api/ping"))

	tokenAuth = jwtauth.New("HS256", key, nil)

	// No auth routes
	apiRouter.Group(func(r chi.Router) {
		r.Get("/", routeRoot)
		r.Post("/login", login)
		r.Get("/login/realms", listRealms) // This is a duplicate of the admin route, but it's necessary for the login.
	})

	// Auth routes
	apiRouter.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator(tokenAuth))

		r.Get("/whoami", whoami)

		r.Get("/vm", vms)
		r.Post("/vm", newVM)

		r.Delete("/vm/{id}", deleteVM)
	})

	// Admin Auth routes
	apiRouter.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(AdminAuthenticator(tokenAuth))

		r.Get("/admin/users", listUsers)
		r.Get("/admin/realms", listRealms)
		r.Get("/admin/realms/{id}", getRealm)
		r.Put("/admin/realms/{id}", updateRealm)
		r.Delete("/admin/realms/{id}", deleteRealm)

		r.Post("/admin/realms", addRealm)
	})

	router.Mount("/api", apiRouter)
}

func ListenAndServe(c config.Server) error {
	if router == nil {
		panic("Router not initialized")
	}

	logger.Info("Listening", "bind", c.Bind)
	return http.ListenAndServe(c.Bind, router)
}

func routeRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the Sasso API!"))
}
