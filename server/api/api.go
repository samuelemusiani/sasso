package api

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"

	"samuelemusiani/sasso/internal/auth"
	"samuelemusiani/sasso/server/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	publicRouter  *chi.Mux     = nil
	privateRouter *chi.Mux     = nil
	logger        *slog.Logger = nil

	tokenAuth *jwtauth.JWTAuth = nil

	publicServerConfig  config.Server
	privateServerConfig config.Server
)

func Init(apiLogger *slog.Logger, key []byte, secret string, frontFS fs.FS, publicServer config.Server, privateServer config.Server) {
	// Logger
	logger = apiLogger

	// Router
	publicRouter = chi.NewRouter()
	privateRouter = chi.NewRouter()

	// Middleware
	publicRouter.Use(middleware.RealIP)
	publicRouter.Use(middleware.Recoverer)
	publicRouter.Use(middleware.CleanPath)

	privateRouter.Use(middleware.RealIP)
	privateRouter.Use(middleware.Recoverer)
	privateRouter.Use(middleware.CleanPath)

	apiRouter := chi.NewRouter()

	if publicServer.LogRequests {
		apiRouter.Use(middleware.Logger)
	}
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(prometheusHandler("/api"))
	apiRouter.Use(middleware.Heartbeat("/api/ping"))

	if privateServer.LogRequests {
		privateRouter.Use(middleware.Logger)
	}
	privateRouter.Use(middleware.Recoverer)
	privateRouter.Use(prometheusHandler("/internal"))
	privateRouter.Use(middleware.Heartbeat("/internal/ping"))

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
			r.Use(validateVMOwnership())

			r.Get("/", getVM)
			r.Delete("/", deleteVM)

			r.Post("/start", changeVMState("start"))
			r.Post("/stop", changeVMState("stop"))
			r.Post("/restart", changeVMState("restart"))

			r.Get("/interface", getInterfaces)
			r.Post("/interface", addInterface)

			r.Route("/interface/{ifaceid}", func(r chi.Router) {
				// Add Interface-specific middleware here (e.g., Interface ownership validation)
				r.Use(validateInterfaceOwnership())

				r.Put("/", updateInterface)
				r.Delete("/", deleteInterface)
			})

			r.Get("/backup", listBackups)
			r.Post("/backup", createBackup)

			r.Delete("/backup/{backupid}", deleteBackup)
			r.Post("/backup/{backupid}/restore", restoreBackup)
			r.Post("/backup/{backupid}/protect", protectBackup)

			r.Get("/backup/request", listBackupRequests)
			r.Get("/backup/request/{requestid}", getBackupRequest)
		})

		r.Post("/net", createNet)
		r.Get("/net", listNets)
		r.Delete("/net/{id}", deleteNet)

		r.Get("/ssh-keys", getSSHKeys)
		r.Post("/ssh-keys", addSSHKey)
		r.Delete("/ssh-keys/{id}", deleteSSHKey)

		r.Get("/vpn", getUserVPNConfig)

		r.Get("/port-forwards", listPortForwards)
		r.Post("/port-forwards", addPortForward)
		r.Delete("/port-forwards/{id}", deletePortForward)

		r.Get("/resources", getUserResources)
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

		r.Get("/admin/port-forwards", listAllPortForwards)
		r.Put("/admin/port-forwards/{id}", approvePortForward)
	})

	// Internal routes
	internalRouter := chi.NewRouter()
	internalRouter.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware(secret))

		r.Get("/net", internalListNets)
		r.Put("/net/{id}", internalUpdateNet)

		r.Get("/vpn", getVPNConfigs)
		r.Put("/vpn", updateVPNConfig)

		r.Get("/user", listUsers)

		r.Get("/port-forwards", internalListProtForwards)
	})

	publicRouter.Mount("/api", apiRouter)
	privateRouter.Mount("/internal", internalRouter)
	privateRouter.Mount("/metrics", promhttp.Handler())

	publicRouter.Get("/*", frontHandler(frontFS))
}

func ListenAndServe() error {
	if publicRouter == nil {
		panic("Router not initialized")
	}
	if privateRouter == nil {
		panic("Router not initialized")
	}

	c := make(chan error, 1)

	go func() {
		logger.Info("Public router listening", "bind", publicServerConfig.Bind)
		err := http.ListenAndServe(publicServerConfig.Bind, publicRouter)
		c <- err
	}()

	go func() {
		logger.Info("Private router listening", "bind", privateServerConfig.Bind)
		err := http.ListenAndServe(privateServerConfig.Bind, privateRouter)
		c <- err
	}()

	return <-c
}

func routeRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the Sasso API!"))
}

func frontHandler(ui_fs fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path[1:]
		if p == "" || p == "static" || p == "static/" {
			p = "index.html"
		}

		f, err := fs.ReadFile(ui_fs, p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
				// If the file does not exists it could be a route that the SPA router
				// would catch. We serve the index.html instead

				f, err = fs.ReadFile(ui_fs, "index.html")
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						http.Error(w, "", http.StatusNotFound)
					} else {
						slog.With("err", err).Error("Reading index.html")
						http.Error(w, "", http.StatusInternalServerError)
					}
					return
				}
				w.Header().Set("Content-Type", "text/html")
				w.Write(f)
				return
			}
			slog.With("path", p, "err", err).Error("Reading file")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		switch path.Ext(p) {
		case ".js":
			w.Header().Set("Content-Type", "text/javascript")
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		}
		w.Write(f)
	}
}
