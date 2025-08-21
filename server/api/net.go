package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type createNetRequest struct {
	Name      string `json:"name"`
	VlanAware bool   `json:"vlanaware"`
}

type returnNet struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	VlanAware bool   `json:"vlanaware"`
}

func createNet(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Failed to get user ID from context", http.StatusUnauthorized)
		return
	}

	var req createNetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Network name is required", http.StatusBadRequest)
		return
	}

	net, err := proxmox.AssignNewNetToUser(userID, req.Name)
	if err != nil {
		if err == proxmox.ErrInsufficientResources {
			http.Error(w, "Insufficient resources", http.StatusForbidden)
		} else {
			http.Error(w, "Failed to create network", http.StatusInternalServerError)
		}
		return
	}

	returnableNet := returnNet{
		ID:     net.ID,
		Name:   net.Alias, // This is correct. For the user the name is the alias.
		Status: net.Status,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(returnableNet)
}

func listNets(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Failed to get user ID from context", http.StatusUnauthorized)
		return
	}

	nets, err := db.GetNetsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to get networks", http.StatusInternalServerError)
		return
	}

	returnableNets := make([]returnNet, len(nets))
	for i, net := range nets {
		returnableNets[i] = returnNet{
			ID:        net.ID,
			Name:      net.Alias,
			Status:    net.Status,
			VlanAware: net.VlanAware,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(returnableNets)
}

func deleteNet(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Failed to get user ID from context", http.StatusUnauthorized)
		return
	}

	netIDStr := chi.URLParam(r, "id")
	netID, err := strconv.ParseUint(netIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid net ID", http.StatusBadRequest)
		return
	}

	if err := proxmox.DeleteNet(userID, uint(netID)); err != nil {
		if err == proxmox.ErrVNetNotFound {
			http.Error(w, "Net not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete net", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
