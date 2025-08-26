package main

import (
	"log/slog"
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/gateway"
	"samuelemusiani/sasso/router/ticket"
	"time"
)

func worker() {
	time.Sleep(5 * time.Second)

	logger := slog.With("module", "worker")
	logger.Info("Worker started")

	gtw := gateway.Get()
	if gtw == nil {
		panic("Gateway not initialized")
	}

	for {
		logger.Debug("Checking for pending tickets")
		ts, err := db.GetTicketsWithStatus("pending")
		if err != nil {
			logger.With("error", err).Error("Failed to get pending tickets from database")
			time.Sleep(10 * time.Second)
			continue
		}

		logger.With("tickets", ts).Debug("Found pending tickets")

		for _, dbt := range ts {
			logger.Info("Processing ticket", "ticket_id", dbt.UUID)
			t, err := ticket.GetTicketByID(dbt.UUID)
			if err != nil {
				logger.With("error", err, "ticket_id", dbt.UUID).Error("Failed to get ticket from database")
				continue
			}

			req := t.GetRequest()
			err = req.Execute(gtw)
			if err != nil {
				logger.With("error", err, "ticket_id", dbt.UUID).Error("Failed to execute ticket request")
				continue
			}
			err = req.SaveToDB(dbt.UUID)
			if err != nil {
				logger.With("error", err, "ticket_id", dbt.UUID).Error("Failed to save request result to database")
				continue
			}
		}

		time.Sleep(5 * time.Second)
	}
}
