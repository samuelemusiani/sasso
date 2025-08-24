package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/router/ticket"

	"github.com/go-chi/chi/v5"
)

type newTicketResponse struct {
	TicketID string `json:"ticket_id"` // ID of the ticket
}

func returnTicket(ticket *ticket.Ticket, w http.ResponseWriter) {
	resp := newTicketResponse{
		TicketID: ticket.GetID().String(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response to JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type ticketResponse struct {
	ID          string `json:"ticket_id"`    // ID of the ticket
	RequestType string `json:"request_type"` // Type of the request associated with the ticket
	Request     any    `json:"request"`      // The request associated with the ticket
}

func requestByTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "ticketID")
	if ticketID == "" {
		http.Error(w, "Ticket ID is required", http.StatusBadRequest)
		return
	}

	t, err := ticket.GetTicketByID(ticketID)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	resp := ticketResponse{
		ID:          t.GetID().String(),
		RequestType: string(t.GetRequest().GetType()),
		Request:     t.GetRequest(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response to JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
