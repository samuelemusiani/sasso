package api

import (
	"log/slog"
	"net/http"

	"samuelemusiani/sasso/server/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var router *chi.Mux = nil
var logger *slog.Logger = nil

func Init(apiLogger *slog.Logger) {
	// Logger
	logger = apiLogger

	// Router
	router = chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Heartbeat("/ping"))

	// Routes
	router.Get("/", routeRoot)
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
