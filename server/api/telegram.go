package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/notify"
	"strconv"

	"github.com/go-chi/chi/v5"
)

var (
	telegramBotTokenRegex  = regexp.MustCompile(`^\d{10}:(\w|-){35}$`)
	telegramBotChatIDRegex = regexp.MustCompile(`^-?\d{5,20}$`)
)

type returnedTelegramBot struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Notes  string `json:"notes"`
	ChatID string `json:"chat_id"`
}

func listTelegramBots(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	bots, err := db.GetTelegramBotsByUserID(userID)
	if err != nil {
		logger.Error("Failed to retrieve telegram bots", "error", err)
		http.Error(w, "Failed to retrieve telegram bots", http.StatusInternalServerError)
		return
	}

	var returnedBots []returnedTelegramBot
	for _, bot := range bots {
		returnedBots = append(returnedBots, returnedTelegramBot{
			ID:     bot.ID,
			Name:   bot.Name,
			Notes:  bot.Notes,
			ChatID: bot.ChatID,
		})
	}

	if err := json.NewEncoder(w).Encode(returnedBots); err != nil {
		logger.Error("Failed to encode telegram bots", "error", err)
		http.Error(w, "Failed to encode telegram bots", http.StatusInternalServerError)
		return
	}
}

type createTelegramBotRequest struct {
	Name   string `json:"name"`
	Notes  string `json:"notes"`
	Token  string `json:"token"`
	ChatID string `json:"chat_id"`
}

func createTelegramBot(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	var req createTelegramBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Token == "" || req.ChatID == "" {
		http.Error(w, "Name, token, and chat_id are required", http.StatusBadRequest)
		return
	}

	if !telegramBotTokenRegex.MatchString(req.Token) {
		http.Error(w, "Invalid Telegram bot token format", http.StatusBadRequest)
		return
	}

	if !telegramBotChatIDRegex.MatchString(req.ChatID) {
		http.Error(w, "Invalid Telegram chat ID format", http.StatusBadRequest)
		return
	}

	err := db.CreateTelegramBot(req.Name, req.Notes, req.Token, req.ChatID, userID)
	if err != nil {
		logger.Error("Failed to create telegram bot", "userID", userID, "error", err)
		http.Error(w, "Failed to create telegram bot", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func deleteTelegramBot(w http.ResponseWriter, r *http.Request) {
	sbotID := chi.URLParam(r, "id")
	botID, err := strconv.ParseUint(sbotID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid bot ID", http.StatusBadRequest)
		return
	}

	userID := mustGetUserIDFromContext(r)
	err = db.DeleteTelegramBot(uint(botID), userID)
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Telegram bot not found", http.StatusNotFound)
		} else {
			logger.Error("Failed to delete telegram bot", "botID", botID, "userID", userID, "error", err)
			http.Error(w, "Failed to delete telegram bot", http.StatusInternalServerError)
		}
		return
	}
}

func testTelegramBot(w http.ResponseWriter, r *http.Request) {
	sbotID := chi.URLParam(r, "id")
	botID, err := strconv.ParseUint(sbotID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid bot ID", http.StatusBadRequest)
		return
	}

	userID := mustGetUserIDFromContext(r)
	bot, err := db.GetTelegramBotByID(uint(botID))
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Telegram bot not found", http.StatusNotFound)
		} else {
			logger.Error("Failed to retrieve telegram bot", "botID", botID, "userID", userID, "error", err)
			http.Error(w, "Failed to retrieve telegram bot", http.StatusInternalServerError)
		}
		return
	}
	if bot.UserID != userID {
		http.Error(w, "Telegram bot not found", http.StatusNotFound)
		return
	}

	err = notify.SendTestBotNotification(bot, "Test notification from Sasso")
	if err != nil {
		logger.Error("Failed to send test notification", "botID", botID, "userID", userID, "error", err)
		http.Error(w, "Failed to send test notification", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
