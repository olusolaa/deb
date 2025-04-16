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
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Context key type for user info
type contextKey string

const userContextKey contextKey = "user"

// UserClaims holds the user information extracted from the JWT
type UserClaims struct {
	UserID   string
	GoogleID string
	Email    string
	Name     string
	Picture  string
}

// APIHandler now includes AuthService, VerseService and JWT secret
type APIHandler struct {
	chatService  service.ChatService
	planService  service.PlanService
	verseService service.VerseService // Add VerseService for on-demand verse content
	authService  *service.AuthService // Add AuthService
	jwtSecret    []byte               // Store JWT secret for middleware
}

// Update NewAPIHandler
func NewAPIHandler(cs service.ChatService, ps service.PlanService, vs service.VerseService, as *service.AuthService, jwtSecret string) *APIHandler {
	return &APIHandler{
		chatService:  cs,
		planService:  ps,
		verseService: vs, // Inject VerseService
		authService:  as, // Inject AuthService
		jwtSecret:    []byte(jwtSecret),
	}
}

// --- Auth Handlers ---

func (h *APIHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// 1. Generate state and store it in a short-lived cookie
	state, err := service.GenerateOauthState()
	if err != nil {
		log.Printf("ERROR: Failed to generate OAuth state: %v", err)
		writeError(w, "Internal server error during login initiation.", http.StatusInternalServerError)
		return
	}

	// Set cookie (HttpOnly, Secure in production, SameSite=Lax is often good for OAuth)
	cookie := http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute), // Short expiry
		HttpOnly: true,
		// Secure:   true, // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)

	// 2. Get the Google login URL and redirect
	loginURL := h.authService.GetGoogleLoginURL(state)
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

func (h *APIHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Check state cookie
	stateCookie, err := r.Cookie("oauthstate")
	if err != nil {
		log.Printf("WARN: Missing oauthstate cookie: %v", err)
		writeError(w, "Invalid state: missing state cookie", http.StatusBadRequest)
		return
	}

	// Clear the state cookie immediately after reading
	http.SetCookie(w, &http.Cookie{
		Name:     "oauthstate",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		// Secure:   true, // Match Secure flag from setting
		SameSite: http.SameSiteLaxMode,
	})

	// 2. Compare state query param with cookie value
	queryState := r.URL.Query().Get("state")
	if queryState == "" || queryState != stateCookie.Value {
		log.Printf("WARN: Invalid OAuth state: cookie='%s' query='%s'", stateCookie.Value, queryState)
		writeError(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// 3. Check for error query param from Google
	queryError := r.URL.Query().Get("error")
	if queryError != "" {
		log.Printf("WARN: Google OAuth error in callback: %s", queryError)
		// Redirect to frontend error page or show generic error
		// For now, redirect to a hypothetical frontend path
		frontendURL := "/login?error=" + url.QueryEscape("Google login failed: "+queryError)
		http.Redirect(w, r, frontendURL, http.StatusSeeOther)
		return
	}

	// 4. Handle the callback code exchange and user lookup/creation
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("WARN: Missing code in Google OAuth callback")
		writeError(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	_, jwtToken, err := h.authService.HandleGoogleCallback(ctx, code)
	if err != nil {
		log.Printf("ERROR: Google callback handling failed: %v", err)
		// Redirect to frontend error page
		frontendURL := "/login?error=" + url.QueryEscape("Authentication processing failed")
		http.Redirect(w, r, frontendURL, http.StatusSeeOther)
		return
	}

	// 5. Success! Set the JWT as a cookie (or return in JSON for frontend to store)
	// Setting as HttpOnly cookie is generally more secure against XSS
	jwtCookie := http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24 * 7), // Match JWT expiry
		HttpOnly: true,
		// Secure:   true, // Set to true if using HTTPS
		SameSite: http.SameSiteLaxMode, // Lax is often suitable
	}
	http.SetCookie(w, &jwtCookie)

	// Redirect the user to the frontend application homepage or dashboard
	// Extract the frontend URL from the CORS allowed origin
	frontendURL := "http://localhost:3000"
	http.Redirect(w, r, frontendURL, http.StatusSeeOther)
}

// HandleLogout clears the auth cookie
func (h *APIHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Expire immediately
		HttpOnly: true,
		// Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// HandleGetCurrentUser returns info about the logged-in user from JWT
func (h *APIHandler) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userClaims, ok := UserFromContext(r.Context())
	if !ok {
		// This should ideally not happen if middleware is applied correctly
		log.Printf("ERROR: User claims not found in context in HandleGetCurrentUser")
		writeError(w, "Authentication token is missing or invalid", http.StatusUnauthorized)
		return
	}

	// Return relevant user info (don't expose everything from JWT if not needed)
	userInfo := map[string]string{
		"id":      userClaims.UserID,
		"email":   userClaims.Email,
		"name":    userClaims.Name,
		"picture": userClaims.Picture,
		// Add other fields as needed by the frontend
	}
	writeJSON(w, http.StatusOK, userInfo)
}

// --- Middleware ---

// AuthMiddleware verifies the JWT token from the cookie or header
func (h *APIHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// 1. Try getting token from HttpOnly cookie first
		cookie, err := r.Cookie("auth_token")
		if err == nil && cookie.Value != "" {
			tokenString = cookie.Value
		} else {
			// 2. If no cookie, try Authorization: Bearer header
			bearerToken := r.Header.Get("Authorization")
			if strings.HasPrefix(bearerToken, "Bearer ") {
				tokenString = strings.TrimPrefix(bearerToken, "Bearer ")
			} else {
				log.Println("DEBUG: No auth cookie or Bearer token found")
				writeError(w, "Authorization token required", http.StatusUnauthorized)
				return
			}
		}

		if tokenString == "" {
			log.Println("DEBUG: Empty token string after checks")
			writeError(w, "Authorization token required", http.StatusUnauthorized)
			return
		}

		// 3. Validate the token
		claims, err := h.authService.ValidateJWT(tokenString)
		if err != nil {
			log.Printf("WARN: Invalid JWT token: %v", err)
			if errors.Is(err, jwt.ErrTokenExpired) {
				writeError(w, "Token has expired", http.StatusUnauthorized)
			} else {
				writeError(w, "Invalid token", http.StatusUnauthorized)
			}
			return
		}

		// 4. Extract user info and add to context
		userID, _ := claims["sub"].(string) // Subject (our internal ID)
		googleID, _ := claims["gid"].(string)
		email, _ := claims["eml"].(string)
		name, _ := claims["nam"].(string)
		picture, _ := claims["pic"].(string)

		if userID == "" {
			log.Printf("ERROR: User ID (sub) missing or invalid in JWT claims: %+v", claims)
			writeError(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		userClaims := UserClaims{
			UserID:   userID,
			GoogleID: googleID,
			Email:    email,
			Name:     name,
			Picture:  picture,
		}
		ctx := context.WithValue(r.Context(), userContextKey, userClaims)

		// 5. Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext retrieves the UserClaims from the request context
func UserFromContext(ctx context.Context) (UserClaims, bool) {
	user, ok := ctx.Value(userContextKey).(UserClaims)
	return user, ok
}

// --- Plan Handlers (Modified for Auth) ---

// CreatePlanRequest remains the same
type CreatePlanRequest struct {
	Topic        string `json:"topic"`
	DurationDays int    `json:"duration_days"`
}

// HandleCreatePlan now associates the plan with the logged-in user
func (h *APIHandler) HandleCreatePlan(w http.ResponseWriter, r *http.Request) {
	userClaims, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, "User authentication failed", http.StatusUnauthorized)
		return
	}

	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Topic == "" || req.DurationDays <= 0 {
		writeError(w, "Topic and positive duration_days are required", http.StatusBadRequest)
		return
	}

	targetAudience := "14-year-old niece" // Still hardcoded

	// Pass the authenticated user's ID to the service
	plan, err := h.planService.CreatePlan(r.Context(), userClaims.UserID, req.Topic, req.DurationDays, targetAudience)
	if err != nil {
		log.Printf("ERROR: Plan creation failed for user %s: %v", userClaims.UserID, err)
		writeError(w, "Failed to create reading plan.", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

// HandleListPlans now lists plans only for the logged-in user
func (h *APIHandler) HandleListPlans(w http.ResponseWriter, r *http.Request) {
	userClaims, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, "User authentication failed", http.StatusUnauthorized)
		return
	}

	// Get plans for this specific user
	plans, err := h.planService.ListPlans(r.Context(), userClaims.UserID)
	if err != nil {
		log.Printf("ERROR: Failed to list plans for user %s: %v", userClaims.UserID, err)
		writeError(w, "Failed to retrieve plan list", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

// HandleGetPlanVerseToday now gets the plan for the logged-in user
func (h *APIHandler) HandleGetPlanVerseToday(w http.ResponseWriter, r *http.Request) {
	userClaims, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, "User authentication failed", http.StatusUnauthorized)
		return
	}

	// Check if plan exists
	verse, err := h.planService.GetActiveVerseForToday(r.Context(), userClaims.UserID)
	if err != nil {
		if err.Error() == "no active reading plan found" {
			writeError(w, "No active reading plan found for today.", http.StatusNotFound)
		} else if errors.Is(err, repository.ErrDayOutOfRange) {
			writeError(w, "Your reading plan is finished!", http.StatusOK)
		} else {
			log.Printf("ERROR: Failed to get today's plan verse for user %s: %v", userClaims.UserID, err)
			writeError(w, "Failed to retrieve today's verse", http.StatusInternalServerError)
		}
		return
	}

	// Check if client explicitly requests no content via query parameter
	skipContent := r.URL.Query().Get("content") == "false"

	if !skipContent {
		// By default, get full verse content using the verse service
		enrichedVerse, err := h.planService.GetEnrichedVerseForToday(r.Context(), userClaims.UserID, h.verseService)
		if err == nil {
			// Return the verse with full content
			writeJSON(w, http.StatusOK, enrichedVerse)
			return
		}

		// Still return the verse with just the reference, but add an error message
		verse.Text = "[Error fetching verse content. Please try again.]"
		writeJSON(w, http.StatusOK, verse)
		return
	}

	// Only return the reference and title if explicitly requested
	writeJSON(w, http.StatusOK, verse)
}

// --- Chat Handlers (Can also be protected) ---

type ChatRequest struct {
	Verse    domain.DailyVerse `json:"verse"`
	Question string            `json:"question"`
}

type ChatResponse struct {
	Answer     string `json:"answer"`
	UsageToday int    `json:"usage_today,omitempty"`
	DailyLimit int    `json:"daily_limit,omitempty"`
}

// HandleChat requires authentication
func (h *APIHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	userClaims, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, "User authentication failed", http.StatusUnauthorized)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Question == "" {
		writeError(w, "Question cannot be empty", http.StatusBadRequest)
		return
	}

	// Pass user ID for rate limiting
	answer, err := h.chatService.GetResponse(r.Context(), req.Verse, req.Question, userClaims.UserID)
	if err != nil {
		log.Printf("ERROR: Failed to get chat response for user %s: %v", userClaims.UserID, err)
		if errors.Is(err, context.DeadlineExceeded) {
			writeError(w, "Chatbot request timed out.", http.StatusGatewayTimeout)
		} else if _, ok := err.(service.ErrRateLimitExceeded); ok {
			// Special case for rate limiting with a friendly message
			writeError(w, "⏰ Daily chat limit reached. Try again tomorrow! We're working on increasing limits soon.", http.StatusTooManyRequests)
		} else {
			writeError(w, "Chatbot couldn't answer right now.", http.StatusInternalServerError)
		}
		return
	}

	// Get current usage stats to include in response
	currentUsage, dailyLimit, _ := h.chatService.GetChatUsage(r.Context(), userClaims.UserID)

	// Include usage information in the response
	writeJSON(w, http.StatusOK, ChatResponse{
		Answer:     answer,
		UsageToday: currentUsage,
		DailyLimit: dailyLimit,
	})
}

// HandleResetChat requires authentication
func (h *APIHandler) HandleResetChat(w http.ResponseWriter, r *http.Request) {
	userClaims, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, "User authentication failed", http.StatusUnauthorized)
		return
	}

	// Pass user context if the chat service needs it (e.g., to clear user-specific history)
	// Example: err := h.chatService.ResetChatHistoryForUser(r.Context(), userClaims.UserID)
	err := h.chatService.ResetChatHistory(r.Context())
	if err != nil {
		log.Printf("ERROR: Failed to reset chat history for user %s: %v", userClaims.UserID, err)
		writeError(w, "Failed to reset chat history", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Chat history reset successfully"})
}

// --- Helper Functions ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("ERROR: Failed to write JSON response: %v", err)
			http.Error(w, `{"error":"Failed to encode response"}`, http.StatusInternalServerError)
		}
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, message string, status int) {
	// Avoid double logging if already logged before calling
	// log.Printf("DEBUG: Writing error response: status=%d, message=%s", status, message)
	writeJSON(w, status, ErrorResponse{Error: message})
}
