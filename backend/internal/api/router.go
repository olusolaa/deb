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

	// --- Base Middleware (applied to all routes) ---
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Consider structured logging for production
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second)) // General request timeout

	// --- CORS Configuration (applied before routing groups) ---
	// Allow requests from the frontend origin, allow credentials (cookies)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{allowedOrigin},
		// Allow specific methods needed by your frontend
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// Allow headers the frontend might send, including Authorization for JWT
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		// Headers the browser is allowed to access in responses
		ExposedHeaders:   []string{"Link"},
		// Crucial for sending/receiving cookies (like the auth_token)
		AllowCredentials: true,
		MaxAge:           86400, // Cache CORS preflight response for 1 day
	}))

	// --- Public Routes ---
	// Health check - doesn't need auth
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Authentication routes - don't need auth middleware applied *before* them
	r.Route("/auth", func(r chi.Router) {
		r.Get("/google/login", h.HandleGoogleLogin)
		r.Get("/google/callback", h.HandleGoogleCallback)
		// Logout might need auth middleware *if* it needs to know *who* is logging out
		// but often just clearing the cookie is enough, so keep it public for simplicity.
		r.Post("/logout", h.HandleLogout)
	})

	// --- Protected API Routes (require authentication) ---
	r.Route("/api", func(r chi.Router) {
		// Apply the AuthMiddleware to all routes within this group
		r.Use(h.AuthMiddleware)

		// Get current user info
		r.Get("/me", h.HandleGetCurrentUser)

		// Reading Plan routes
		r.Route("/plans", func(r chi.Router) {
			r.Post("/", h.HandleCreatePlan)       // POST /api/plans
			r.Get("/", h.HandleListPlans)          // GET /api/plans
			r.Get("/today", h.HandleGetPlanVerseToday) // GET /api/plans/today
			// Add routes for getting a specific plan? PUT/DELETE plans?
			// r.Get("/{planID}", h.HandleGetPlan)
			// r.Put("/{planID}", h.HandleUpdatePlan)
			// r.Delete("/{planID}", h.HandleDeletePlan)
		})

		// Chat routes
		r.Route("/chat", func(r chi.Router) {
			r.Post("/", h.HandleChat)             // POST /api/chat
			r.Post("/reset", h.HandleResetChat)   // POST /api/chat/reset
		})

		// NOTE: The previous `/api/admin` group is removed as we now
		// integrate plan creation/listing into the main `/api/plans`
		// protected by the AuthMiddleware. Access control (admin vs user)
		// would typically be handled *within* the handlers based on user roles/permissions
		// if needed later.
	})

	return r
}
