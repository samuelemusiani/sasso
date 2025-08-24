package ticket

import (
	"log/slog"
	"samuelemusiani/sasso/router/db"

	"github.com/google/uuid"
)

type Ticket struct {
	id      uuid.UUID
	request Request
}

func NewTicket() *Ticket {
	return &Ticket{
		id: uuid.Must(uuid.NewV7()),
	}
}

func NewTicketWithRequest(req Request) *Ticket {
	return &Ticket{
		id:      uuid.Must(uuid.NewV7()),
		request: req,
	}
}

func (t *Ticket) SetRequest(req Request) {
	t.request = req
}

func (t *Ticket) GetRequest() Request {
	return t.request
}

func (t *Ticket) GetID() uuid.UUID {
	return t.id
}

func (t *Ticket) SaveToDB() error {
	return t.request.SaveToDB(t.id.String())
}

func GetTicketByID(ticketID string) (*Ticket, error) {
	id, err := uuid.Parse(ticketID)
	if err != nil {
		slog.With("uuid", ticketID, "err", err).Error("Invalid ticket ID format")
		return nil, err
	}

	dbt, err := db.GetTicketByUUID(id.String())
	if err != nil {
		slog.With("uuid", ticketID, "err", err).Error("Failed to retrieve ticket by ID")
		return nil, err
	}

	req, err := requestFromDBByTicket(dbt)
	if err != nil {
		slog.With("uuid", ticketID, "err", err).Error("Failed to retrieve request from database by ticket")
		return nil, err
	}

	t := Ticket{
		id:      id,
		request: req,
	}

	return &t, nil
}
