package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	router *chi.Mux
)

type NewNet struct {
	Subnet string `json:"subnet"`
}

func Init() {
	router = chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.CleanPath)

	router.Get("/", helloHandler)
	router.Post("/api", apiHandler)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}

func apiHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Received body: %s\n", string(body))

	var newNet NewNet
	err = json.Unmarshal(body, &newNet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Parsed JSON: %+v\n", newNet)
	w.Write([]byte("Api, hello!"))
}

func ListenAndServe(bind string) error {
	slog.Info("Listening on: ", "bind", bind)
	return http.ListenAndServe(bind, router)
}
