package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"samuelemusiani/sasso/internal/auth"
	"samuelemusiani/sasso/server/config"
)

var (
	publicRouter  *chi.Mux
	privateRouter *chi.Mux
	logger        *slog.Logger

	tokenAuth *jwtauth.JWTAuth

	privateServer *http.Server
	publicServer  *http.Server
	portForwards  config.PortForwards
	vpnConfigs    config.VPN
)

func Init(apiLogger *slog.Logger, key []byte, secret string, frontFS fs.FS, publicServerConf config.Server, privateServerConf config.Server, pf config.PortForwards, vpn config.VPN) error {
	// Logger
	logger = apiLogger

	if err := checkConfig(key, secret, publicServerConf, privateServerConf, pf, vpn); err != nil {
		return err
	}

	// Router
	publicRouter = chi.NewRouter()
	privateRouter = chi.NewRouter()

	// Servers
	publicServer = &http.Server{
		Addr:    publicServerConf.Bind,
		Handler: publicRouter,
	}

	privateServer = &http.Server{
		Addr:    privateServerConf.Bind,
		Handler: privateRouter,
	}

	portForwards = pf
	vpnConfigs = vpn

	// Middleware
	publicRouter.Use(middleware.RealIP)
	publicRouter.Use(middleware.Recoverer)
	publicRouter.Use(middleware.CleanPath)

	privateRouter.Use(middleware.RealIP)
	privateRouter.Use(middleware.Recoverer)
	privateRouter.Use(middleware.CleanPath)

	apiRouter := chi.NewRouter()

	if publicServerConf.LogRequests {
		apiRouter.Use(middleware.Logger)
	}

	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(prometheusHandler("/api"))
	apiRouter.Use(middleware.Heartbeat("/api/ping"))

	if privateServerConf.LogRequests {
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

			r.Get("/interface", getInterfacesForVM)
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

			r.Patch("/lifetime", updateVMLifetime)
			r.Patch("/resources", updateVMResources)
		})

		r.Post("/net", createNet)
		r.Get("/net", listNets)
		r.Put("/net/{id}", updateNet)
		r.Delete("/net/{id}", deleteNet)

		r.Get("/interfaces", getAllInterfaces)

		r.Get("/ssh-keys", getSSHKeys)
		r.Post("/ssh-keys", addSSHKey)
		r.Delete("/ssh-keys/{id}", deleteSSHKey)

		r.Get("/vpn", getUserVPNConfigs)
		r.Post("/vpn", addVPNConfig)
		r.Delete("/vpn/{id}", deleteVPNConfig)

		r.Get("/port-forwards/public-ip", func(w http.ResponseWriter, _ *http.Request) {
			if err := json.NewEncoder(w).Encode(map[string]string{"public_ip": portForwards.PublicIP}); err != nil {
				slog.Error("marshaling public IP", "err", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}
		})

		r.Get("/port-forwards", listPortForwards)
		r.Post("/port-forwards", addPortForward)
		r.Delete("/port-forwards/{id}", deletePortForward)

		r.Get("/resources", getUserResources)

		r.Get("/notify/telegram", listTelegramBots)
		r.Post("/notify/telegram", createTelegramBot)
		r.Patch("/notify/telegram/{id}", enableDisableTelegramBot)
		r.Delete("/notify/telegram/{id}", deleteTelegramBot)
		r.Post("/notify/telegram/{id}/test", testTelegramBot)

		r.Get("/groups", listUserGroups)
		r.Post("/groups", createGroup)

		// This route does not require group ownership, as it's used to accept invitations
		r.Get("/groups/invites", listGroupInvitations)
		r.Patch("/groups/invites/{inviteid}", manageInvitation)

		r.Route("/groups/{groupid}", func(r chi.Router) {
			r.Use(validateGroupOwnership())
			r.Get("/", getGroup)
			r.Put("/", updateGroup)
			r.Delete("/", deleteGroup)

			// Invitations

			r.Get("/invites", getGroupPendingInvitations)
			r.Post("/invites", inviteUserToGroup)
			r.Delete("/invites/{inviteid}", revokeGroupInvitation)

			// Members management
			r.Get("/members", listGroupMembers)
			r.Get("/members/me", getMyGroupMembership)
			r.Delete("/members/me", leaveGroup)
			r.Delete("/members/{userid}", removeUserFromGroup)

			// Resources management
			r.Get("/resources", getGroupResources)
			r.Post("/resources", addGroupResources)
			r.Put("/resources", modifyGroupResources)
			r.Delete("/resources", revokeGroupResources)
		})

		r.Post("/ip-check", checkIfIPInUse)

		r.Get("/settings", getUserSettings)
		r.Put("/settings", updateUserSettings)
	})

	// Admin Auth routes
	apiRouter.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(AdminAuthenticator())

		r.Get("/admin/users", internalListUsers)
		r.Get("/admin/users/{id}", getUser)

		r.Get("/admin/groups", adminListGroups)
		r.Get("/admin/groups/{id}", adminGetGroup)
		r.Put("/admin/groups/{id}/resources", adminUpdateGroupResources)

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

		r.Get("/vpn", internalGetVPNConfigs)
		r.Put("/vpn", internalUpdateVPNConfig)

		r.Get("/user", internalListUsers)

		r.Get("/port-forwards", internalListProtForwards)
	})

	publicRouter.Mount("/api", apiRouter)
	privateRouter.Mount("/internal", internalRouter)
	privateRouter.Mount("/metrics", promhttp.Handler())

	publicRouter.Get("/*", frontHandler(frontFS))

	return nil
}

func ListenAndServe() chan error {
	if publicRouter == nil {
		panic("Router not initialized")
	}

	if privateRouter == nil {
		panic("Router not initialized")
	}

	c := make(chan error, 1)

	go func() {
		logger.Info("Public router listening", "bind", publicServer.Addr)

		err := publicServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("public server error", "err", err)

			c <- err
		}
	}()

	go func() {
		logger.Info("Private router listening", "bind", privateServer.Addr)

		err := privateServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("private server error", "err", err)

			c <- err
		}
	}()

	return c
}

func Shutdown(publicServerCtx, privateServerCtx context.Context) error {
	c := make(chan error, 2)

	go func() {
		logger.Info("Shutting down public server...")

		err := publicServer.Shutdown(publicServerCtx)
		if err != nil {
			slog.Error("public server shutdown failed", "err", err)
		} else {
			logger.Info("public server shut down")
		}

		c <- err
	}()

	go func() {
		logger.Info("Shutting down private server...")

		err := privateServer.Shutdown(privateServerCtx)
		if err != nil {
			slog.Error("private server shutdown failed", "err", err)
		} else {
			logger.Info("private server shut down")
		}

		c <- err
	}()

	return errors.Join(<-c, <-c)
}

func routeRoot(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("Welcome to the Sasso API!"))
	if err != nil {
		slog.Error("writing response", "err", err)
		http.Error(w, "internal Server Error", http.StatusInternalServerError)
	}
}

func frontHandler(uiFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path[1:]
		if p == "" || p == "static" || p == "static/" {
			p = "index.html"
		}

		f, err := fs.ReadFile(uiFS, p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
				// If the file does not exists it could be a route that the SPA router
				// would catch. We serve the index.html instead
				f, err = fs.ReadFile(uiFS, "index.html")
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						http.Error(w, "", http.StatusNotFound)
					} else {
						slog.Error("reading index.html", "err", err)
						http.Error(w, "", http.StatusInternalServerError)
					}

					return
				}

				w.Header().Set("Content-Type", "text/html")

				_, err = w.Write(f)
				if err != nil {
					slog.Error("writing response", "err", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}

				return
			}

			slog.Error("reading file", "path", p, "err", err)
			http.Error(w, "", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(p)))

		_, err = w.Write(f)
		if err != nil {
			slog.Error("writing response", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			return
		}
	}
}

func checkConfig(key []byte, secret string, publicServerConf config.Server, privateServerConf config.Server, portForwards config.PortForwards, vpn config.VPN) error {
	if len(key) == 0 {
		return errors.New("api key cannot be empty")
	}

	if secret == "" {
		return errors.New("internal secret cannot be empty")
	}

	if publicServerConf.Bind == "" {
		return errors.New("public server bind address cannot be empty")
	}

	if privateServerConf.Bind == "" {
		return errors.New("private server bind address cannot be empty")
	}

	if portForwards.PublicIP == "" {
		return errors.New("port forwards public IP cannot be empty")
	}

	if portForwards.MinPort > portForwards.MaxPort {
		return errors.New("port forwards min port cannot be greater than max port")
	}

	if portForwards.MinPort == 0 || portForwards.MaxPort == 0 {
		return errors.New("port forwards min and max port cannot be zero")
	}

	if portForwards.MaxPort-portForwards.MinPort < 10 {
		return errors.New("port forwards range must be at least 10 ports")
	}

	if vpn.MaxProfilesPerUser == 0 {
		return errors.New("vpn max profiles per user cannot be zero")
	}

	return nil
}
