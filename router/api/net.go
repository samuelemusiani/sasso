package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/router/ticket"
)

// Every time a new network must be create, a request of this type is sent to
// the API.
type newNetRequest struct {
	VNet   string `json:"vnet"`    // Name of the new VNet
	VNetID uint   `json:"vnet_id"` // ID of the new VNet (VXLAN ID)
}

// type newNetResponse struct {
// 	Success bool   `json:"success"` // True if the request was successful
// 	Error   string `json:"error"`   // Error message if the request failed
//
// 	Subnet    string `json:"subnet"`    // Subnet of the new VNet
// 	RouterIP  string `json:"router_ip"` // Router IP of the new VNet
// 	Broadcast string `json:"broadcast"` // Broadcast address of the new VNet
// }

func newNet(w http.ResponseWriter, r *http.Request) {
	var req newNetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	netRequest := ticket.NewNetworkRequest(req.VNet, req.VNetID)
	t := ticket.NewTicketWithRequest(&netRequest)
	err := t.SaveToDB()
	if err != nil {
		http.Error(w, "Failed to save ticket to database", http.StatusInternalServerError)
		return
	}

	returnTicket(t, w)
}

type deleteNetRequest struct {
	VNet   string `json:"vnet"`
	VNetID uint   `json:"vnet_id"`
}

func deleteNet(w http.ResponseWriter, r *http.Request) {
	var req deleteNetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.VNet == "" && req.VNetID == 0 {
		http.Error(w, "Either vnet or vnet_id must be provided", http.StatusBadRequest)
		return
	}

	netRequest := ticket.NewDeleteNetworkRequest(req.VNet, req.VNetID)
	t := ticket.NewTicketWithRequest(&netRequest)
	err := t.SaveToDB()
	if err != nil {
		http.Error(w, "Failed to save ticket to database", http.StatusInternalServerError)
		return
	}

	returnTicket(t, w)
}
