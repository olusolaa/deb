package main

import (
	"bibleapp/backend/internal/api"
	"bibleapp/backend/internal/config"
	"bibleapp/backend/internal/llm"
	"bibleapp/backend/internal/repository"
	"bibleapp/backend/internal/service"
	"context"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func main() {
	// Load Configuration
	cfg := config.Load()

	// --- Dependency Injection ---

	// 1. Set up MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Direct connection to MongoDB using the driver
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		log.Fatalf("FATAL: Could not connect to MongoDB: %v", err)
	}

	// Ping MongoDB to verify connection
	if err = mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("FATAL: Could not ping MongoDB: %v", err)
	}

	defer func() {
		if err = mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("ERROR: Failed to disconnect MongoDB client: %v", err)
		}
	}()

	mongoDB := mongoClient.Database("bibleapp")

	// 2. Repositories
	planRepo := repository.NewMongoPlanRepository(mongoDB)
	userRepo := repository.NewMongoUserRepository(mongoDB)

	// 3. External Clients (LLM)
	planningModelName := cfg.LLMModelName
	openRouterClient := llm.NewOpenRouterClient(cfg.OpenRouterAPIKey, cfg.OpenRouterBaseURL)

	// 4. Services
	// Create verse repository that uses LLM to fetch verse content on-demand
	verseRepo := repository.NewLLMVerseRepository(openRouterClient, cfg.LLMModelName)

	// Create all services
	chatService := service.NewChatService(openRouterClient, cfg.LLMModelName)
	verseService := service.NewVerseService(verseRepo)
	planService := service.NewPlanService(planRepo, openRouterClient, planningModelName)

	// Set up Google OAuth config
	googleOAuthConfig := service.SetupGoogleOAuthConfig(cfg)

	// Create auth service with proper dependencies
	authService := service.NewAuthService(googleOAuthConfig, userRepo, cfg.JWTSecret) // Auth service for Google OAuth

	// 4. API Handler (Inject all services)
	apiHandler := api.NewAPIHandler(chatService, planService, verseService, authService, cfg.JWTSecret)

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
