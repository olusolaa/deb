package main

import (
	"bibleapp/backend/internal/api"
	"bibleapp/backend/internal/config"
	"bibleapp/backend/internal/llm"
	"bibleapp/backend/internal/repository"
	"bibleapp/backend/internal/service"
	"log"
	"net/http"
	// "time" // Needed if adding server timeouts
)

func main() {
	// Load Configuration
	cfg := config.Load()

	// --- Dependency Injection ---

	// 1. Repositories
	planRepo := repository.NewInMemoryPlanRepository() // Use in-memory repo

	// 2. External Clients (LLM)
	// Use a more capable model for planning if configured, else default
	planningModelName := cfg.LLMModelName // Maybe add a specific PLANNING_LLM_MODEL_NAME config later
	openRouterClient := llm.NewOpenRouterClient(cfg.OpenRouterAPIKey, cfg.OpenRouterBaseURL)

	// 3. Services
	chatService := service.NewChatService(openRouterClient, cfg.LLMModelName) // Chat can use default model
	planService := service.NewPlanService(planRepo, openRouterClient)         // Plan service needs repo and LLM

	// 4. API Handler (Inject PlanService, remove old VerseService if unused)
	apiHandler := api.NewAPIHandler(chatService, planService)

	// 5. Router
	router := api.NewRouter(apiHandler, cfg.CorsAllowedOrigin)

	// --- Start Server ---
	serverAddr := ":" + cfg.Port
	log.Printf("INFO: Starting backend server on %s", serverAddr)
	log.Printf("INFO: Allowing CORS requests from: %s", cfg.CorsAllowedOrigin)
	log.Printf("INFO: Using default LLM model for chat: %s", cfg.LLMModelName)
	log.Printf("INFO: Using LLM model for planning: %s", planningModelName) // Log planning model if different

	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
		// Add timeouts
		// ReadTimeout:  15 * time.Second, // Increase slightly for potential LLM delays
		// WriteTimeout: 30 * time.Second,
		// IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("FATAL: Could not listen on %s: %v\n", serverAddr, err)
	}

	log.Println("INFO: Server stopped gracefully.")
}
