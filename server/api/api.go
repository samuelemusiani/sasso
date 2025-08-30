package api

import (
	"log/slog"
	"net/http"

	"samuelemusiani/sasso/server/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
)

var (
	router *chi.Mux     = nil
	logger *slog.Logger = nil

	tokenAuth *jwtauth.JWTAuth = nil
)

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

		// Group VM-specific endpoints with additional middleware
		r.Route("/vm/{vmid}", func(r chi.Router) {
			// Add VM-specific middleware here (e.g., VM ownership validation)
			r.Use(validateVMOwnership())

			r.Delete("/", deleteVM)

			r.Get("/interface", getInterfaces)
			r.Post("/interface", addInterface)

			r.Route("/interface/{ifaceid}", func(r chi.Router) {
				// Add Interface-specific middleware here (e.g., Interface ownership validation)
				r.Use(validateInterfaceOwnership())

				r.Put("/", updateInterface)
				r.Delete("/", deleteInterface)
			})
		})

		r.Post("/net", createNet)
		r.Get("/net", listNets)
		r.Delete("/net/{id}", deleteNet)

		r.Get("/ssh-keys", getSSHKeys)
		r.Post("/ssh-keys", addSSHKey)
		r.Delete("/ssh-keys/{id}", deleteSSHKey)
	})

	// Admin Auth routes
	apiRouter.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(AdminAuthenticator(tokenAuth))

		r.Get("/admin/users", listUsers)
		r.Get("/admin/users/{id}", getUser)
		r.Get("/admin/realms", listRealms)
		r.Get("/admin/realms/{id}", getRealm)
		r.Put("/admin/realms/{id}", updateRealm)
		r.Delete("/admin/realms/{id}", deleteRealm)

		r.Post("/admin/realms", addRealm)
		r.Put("/admin/users/limits", updateUserLimits)

		r.Get("/admin/ssh-keys/global", getGlobalSSHKeys)
		r.Post("/admin/ssh-keys/global", addGlobalSSHKey)
		r.Delete("/admin/ssh-keys/global/{id}", deleteGlobalSSHKey)
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
