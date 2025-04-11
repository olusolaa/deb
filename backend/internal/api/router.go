package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(h *APIHandler, allowedOrigin string) http.Handler {
	r := chi.NewRouter()

	// --- Middleware ---
	r.Use(middleware.RequestID)                 // Add request ID to logs/context
	r.Use(middleware.RealIP)                    // Use X-Forwarded-For or X-Real-IP
	r.Use(middleware.Logger)                    // Log requests (consider structured logging)
	r.Use(middleware.Recoverer)                 // Recover from panics
	r.Use(middleware.Timeout(60 * time.Second)) // Set request timeout

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin}, // Use config value
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Important for cookies, authorization headers
		MaxAge:           300,  // Maximum value not ignored by any of major browsers
	}))

	r.Get("/api/plans/today", h.HandleGetPlanVerseToday) // Renamed endpoint
	r.Post("/api/chat", h.HandleChat)

	// --- Admin Routes (Could be grouped under /api/admin) ---
	r.Route("/api/admin", func(r chi.Router) {
		// Add admin-specific middleware here later (e.g., authentication)
		r.Post("/plans", h.HandleCreatePlan)
		r.Get("/plans", h.HandleListPlans)
	})

	// Optional: Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return r
}
