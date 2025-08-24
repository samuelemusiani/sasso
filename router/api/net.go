package api

import (
	"encoding/json"
	"net/http"
)

// Every time a new network must be create, a request of this type is sent to
// the API.
type newNetRequest struct {
	VNet   string `json:"vnet"`    // Name of the new VNet
	VNetID uint   `json:"vnet_id"` // ID of the new VNet (VXLAN ID)
}

type newNetResponse struct {
	Success bool   `json:"success"` // True if the request was successful
	Error   string `json:"error"`   // Error message if the request failed

	Subnet    string `json:"subnet"`    // Subnet of the new VNet
	RouterIP  string `json:"router_ip"` // Router IP of the new VNet
	Broadcast string `json:"broadcast"` // Broadcast address of the new VNet
}

func newNet(w http.ResponseWriter, r *http.Request) {
	var req newNetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Implement logic

	var resp newNetResponse
	resp.Success = true
	resp.Subnet = "192.168.0.0/27"  // Example subnet
	resp.RouterIP = "192.168.0.30"  // Example router IP
	resp.Broadcast = "192.168.0.31" // Example broadcast address

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response to JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
