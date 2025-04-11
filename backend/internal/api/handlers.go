package api

import (
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/repository" // Import repository for errors
	"bibleapp/backend/internal/service"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// Add PlanService to APIHandler dependencies
type APIHandler struct {
	verseService service.VerseService // Keep if needed for other things? Remove if not.
	chatService  service.ChatService
	planService  service.PlanService // Add PlanService
}

// Update NewAPIHandler
func NewAPIHandler(cs service.ChatService, ps service.PlanService /* remove vs if unused */) *APIHandler {
	return &APIHandler{
		chatService: cs,
		planService: ps, // Inject PlanService
	}
}

type CreatePlanRequest struct {
	Topic        string `json:"topic"`
	DurationDays int    `json:"duration_days"`
	// TargetAudience is fixed for now, could be added to request later
}

type ChatResponse struct {
	Answer string `json:"answer"`
}

// --- Modified/New Handlers ---

func (h *APIHandler) HandleGetPlanVerseToday(w http.ResponseWriter, r *http.Request) {
	verse, err := h.planService.GetActiveVerseForToday(r.Context())
	if err != nil {
		if errors.Is(err, repository.ErrNoActivePlan) {
			writeError(w, "No reading plan is currently active.", http.StatusNotFound)
		} else if errors.Is(err, repository.ErrDayOutOfRange) {
			// Use ErrDayOutOfRange or a dedicated ErrPlanFinished
			writeError(w, "The current reading plan has finished.", http.StatusNotFound)
		} else {
			log.Printf("ERROR: Failed to get today's plan verse: %v", err)
			writeError(w, "Failed to retrieve today's verse from the plan", http.StatusInternalServerError)
		}
		return
	}
	// Return the DailyVerse object which includes reference, text, explanation
	writeJSON(w, http.StatusOK, verse)
}

func (h *APIHandler) HandleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Topic == "" || req.DurationDays <= 0 {
		writeError(w, "Topic and positive duration_days are required", http.StatusBadRequest)
		return
	}

	// Hardcode target audience for now
	targetAudience := "14-year-old niece"

	// Call the service to generate and save the plan
	// This can take a while due to the LLM call! Consider async operations for production.
	plan, err := h.planService.CreatePlan(r.Context(), req.Topic, req.DurationDays, targetAudience)
	if err != nil {
		log.Printf("ERROR: Plan creation failed: %v", err)
		// Check for specific LLM errors if possible
		writeError(w, "Failed to create reading plan. The LLM might be unavailable or the request too complex.", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, plan) // Return the created plan details
}

// New Handler for Listing Plans (Admin)
func (h *APIHandler) HandleListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.planService.ListPlans(r.Context())
	if err != nil {
		log.Printf("ERROR: Failed to list plans: %v", err)
		writeError(w, "Failed to retrieve plan list", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

// HandleChat remains largely the same, but the frontend will now send a DailyVerse object
// Modify ChatRequest or how the handler gets the verse if needed.
// Let's assume the frontend sends the currently displayed DailyVerse info.

type ChatRequest struct {
	// Change Verse from BibleVerse to DailyVerse
	Verse    domain.DailyVerse `json:"verse"`
	Question string            `json:"question"`
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
	// Basic validation for verse data from DailyVerse
	if req.Verse.Text == "" || req.Verse.Reference == "" {
		writeError(w, "Verse text and reference are required", http.StatusBadRequest)
		return
	}

	// Create a temporary BibleVerse struct from DailyVerse for the existing chat service
	// OR modify the chat service to accept DailyVerse / Reference+Text directly.
	// Let's adapt here for simplicity:
	tempBibleVerse := domain.BibleVerse{
		Reference: req.Verse.Reference,
		Text:      req.Verse.Text,
		// Book, Chapter, VerseNumber aren't strictly needed by the current chat prompt
	}

	answer, err := h.chatService.GetResponse(r.Context(), tempBibleVerse, req.Question) // Pass the adapted verse
	if err != nil {
		log.Printf("ERROR: Failed to get chat response: %v", err)
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
