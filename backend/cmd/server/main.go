package main

import (
	"log"
	"net/http"

	"bibleapp/backend/internal/api"
	"bibleapp/backend/internal/config"
	"bibleapp/backend/internal/llm"
	"bibleapp/backend/internal/repository"
	"bibleapp/backend/internal/service"
)

func main() {
	// Load Configuration
	cfg := config.Load()

	// --- Dependency Injection ---

	// 1. Repositories
	verseRepo := repository.NewStaticVerseRepository()

	// 2. External Clients (LLM)
	openRouterClient := llm.NewOpenRouterClient(cfg.OpenRouterAPIKey, cfg.OpenRouterBaseURL)

	// 3. Services
	verseService := service.NewVerseService(verseRepo)
	chatService := service.NewChatService(openRouterClient, cfg.LLMModelName) // Pass LLM client and model name

	// 4. API Handler
	apiHandler := api.NewAPIHandler(verseService, chatService)

	// 5. Router
	router := api.NewRouter(apiHandler, cfg.CorsAllowedOrigin) // Pass allowed origin

	// --- Start Server ---
	serverAddr := ":" + cfg.Port
	log.Printf("INFO: Starting backend server on %s", serverAddr)
	log.Printf("INFO: Allowing CORS requests from: %s", cfg.CorsAllowedOrigin)
	log.Printf("INFO: Using LLM model via OpenRouter: %s", cfg.LLMModelName)

	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
		// Add timeouts for production robustness
		// ReadTimeout:  5 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("FATAL: Could not listen on %s: %v\n", serverAddr, err)
	}

	log.Println("INFO: Server stopped gracefully.")
}
