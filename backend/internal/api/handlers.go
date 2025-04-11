package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/service"
)

// APIHandler holds dependencies for API handlers.
type APIHandler struct {
	verseService service.VerseService
	chatService  service.ChatService
}

// NewAPIHandler creates a new handler instance.
func NewAPIHandler(vs service.VerseService, cs service.ChatService) *APIHandler {
	return &APIHandler{
		verseService: vs,
		chatService:  cs,
	}
}

// --- DTOs (Data Transfer Objects) ---

type ChatRequest struct {
	Verse    domain.BibleVerse `json:"verse"`
	Question string            `json:"question"`
}

type ChatResponse struct {
	Answer string `json:"answer"`
}

// --- Handlers ---

func (h *APIHandler) HandleGetVerseToday(w http.ResponseWriter, r *http.Request) {
	verse, err := h.verseService.GetDailyVerse(r.Context())
	if err != nil {
		log.Printf("ERROR: Failed to get daily verse: %v", err)
		writeError(w, "Failed to retrieve today's verse", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, verse)
}

func (h *APIHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		writeError(w, "Question cannot be empty", http.StatusBadRequest)
		return
	}
	// Basic validation for verse data
	if req.Verse.Text == "" || req.Verse.Reference == "" {
		writeError(w, "Verse text and reference are required", http.StatusBadRequest)
		return
	}

	answer, err := h.chatService.GetResponse(r.Context(), req.Verse, req.Question)
	if err != nil {
		log.Printf("ERROR: Failed to get chat response: %v", err)
		// Distinguish between client errors (bad input - unlikely here now) and server errors (LLM failed)
		if errors.Is(err, context.DeadlineExceeded) {
			writeError(w, "Chatbot request timed out, please try again.", http.StatusGatewayTimeout)
		} else {
			writeError(w, "Chatbot couldn't answer right now.", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, ChatResponse{Answer: answer})
}

// --- Helper Functions ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("ERROR: Failed to write JSON response: %v", err)
			// Attempt to write a plain text error if JSON encoding fails
			http.Error(w, `{"error":"Failed to encode response"}`, http.StatusInternalServerError)
		}
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
