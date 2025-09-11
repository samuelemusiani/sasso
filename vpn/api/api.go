package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/util"
	"samuelemusiani/sasso/vpn/wg"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	shorewall "github.com/samuelemusiani/go-shorewall"
)

var (
	router *chi.Mux
)

type NewNet struct {
	Subnet string `json:"subnet"`
}

type NewAddress struct {
	Address string
	// Address string `json:"address"`
}

type ApiResponse struct {
	Config string `json:"config"`
}

var (
	vpnZone   string
	sassoZone string
)

func Init(config *config.Firewall) {
	router = chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.CleanPath)

	router.Get("/", helloHandler)
	router.Post("/api", apiHandler)

	vpnZone = config.VPNZone
	sassoZone = config.SassoZone
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

	// controllo che la subnet non esista già
	exist, err := db.CheckSubnetExists(newNet.Subnet)
	if err != nil {
		http.Error(w, "Error checking subnet existence", http.StatusInternalServerError)
		return
	}
	if exist {
		http.Error(w, "Subnet already exists", http.StatusConflict)
		return
	}

	// genero indirizzo
	var newAddress NewAddress
	util.Init(slog.Default(), newNet.Subnet)
	newAddress.Address, err = util.NextAvailableAddress()
	if err != nil {
		http.Error(w, "Error generating address", http.StatusInternalServerError)
		return
	}

	// genero config wgInterface
	wgInterface, err := wg.NewWGConfig(newAddress.Address, newNet.Subnet)
	if err != nil {
		http.Error(w, "Error generating WireGuard config", http.StatusInternalServerError)
		return
	}

	// salvo nel db
	err = db.NewInterface(wgInterface.PrivateKey, wgInterface.PublicKey, newNet.Subnet, newAddress.Address)
	if err != nil {
		http.Error(w, "Error saving interface to database", http.StatusInternalServerError)
		return
	}

	// aggiorno firewall
	err = shorewall.AddRule(shorewall.Rule{
		Action:      "ACCEPT",
		Source:      fmt.Sprintf("%s:%s", vpnZone, newAddress.Address),
		Destination: fmt.Sprintf("%s:%s", sassoZone, newNet.Subnet)})
	if err != nil {
		http.Error(w, "Error adding firewall rule", http.StatusInternalServerError)
		return
	}
	if err = shorewall.Reload(); err != nil {
		http.Error(w, "Error reloading firewall", http.StatusInternalServerError)
		return
	}

	// creo interfaccia wg sul server
	err = wg.CreateInterface(wgInterface)
	if err != nil {
		http.Error(w, "Error creating WireGuard interface", http.StatusInternalServerError)
		return
	}

	// ritorno config al client
	apiResp := ApiResponse{Config: wgInterface.String()}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(apiResp)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func ListenAndServe(bind string) error {
	slog.Info("Listening on: ", "bind", bind)
	return http.ListenAndServe(bind, router)
}
