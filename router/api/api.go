package api

import (
	"errors"
	"log/slog"
	"net/http"
	"samuelemusiani/sasso/router/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	router *chi.Mux     = nil
	logger *slog.Logger = nil

	secret string
)

func Init(apiLogger *slog.Logger, apiSecret string) error {
	// Logger
	logger = apiLogger

	if len(apiSecret) < 8 {
		return errors.New("API secret must be at least 8 characters long")
	}
	secret = apiSecret

	// Router
	router = chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.CleanPath)

	apiRouter := chi.NewRouter()

	apiRouter.Use(middleware.Heartbeat("/api/ping"))

	apiRouter.Group(func(r chi.Router) {
		r.Get("/", routeRoot)
	})

	// Auth routes
	apiRouter.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/net", newNet)
		r.Delete("/net", deleteNet)

		r.Get("/ticket/{ticketID}", requestByTicket)
	})

	router.Mount("/api", apiRouter)

	return nil
}

func ListenAndServe(c config.Server) error {
	if router == nil {
		panic("Router not initialized")
	}

	logger.Info("Listening", "bind", c.Bind)
	return http.ListenAndServe(c.Bind, router)
}

func routeRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the sasso-router API!"))
}
