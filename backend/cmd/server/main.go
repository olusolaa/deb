package main

import (
	"bibleapp/backend/internal/api"
	"bibleapp/backend/internal/config"
	"bibleapp/backend/internal/llm"
	"bibleapp/backend/internal/repository"
	"bibleapp/backend/internal/service"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BibleVerse defines the structure for storing Bible verses in MongoDB
type BibleVerse struct {
	Book        string `bson:"book"`
	BookIndex   int    `bson:"book_index"` // Added book index for potential ordering
	Chapter     int    `bson:"chapter"`
	Verse       int    `bson:"verse"`
	Text        string `bson:"text"`
	Translation string `bson:"translation"`
}

// Structs to parse the nested JSON structure from the Bible repository
type BibleJSON struct {
	Book []BookData `json:"Book"`
}

type BookData struct {
	Chapter []ChapterData `json:"Chapter"`
	Name    string        `json:"name,omitempty"`
}

type ChapterData struct {
	Verse   []VerseData `json:"Verse"`
	Chapter string      `json:"chapter,omitempty"`
}

type VerseData struct {
	Verseid string `json:"Verseid"`
	Verse   string `json:"Verse"`
}

const bibleJsonURL = "https://raw.githubusercontent.com/godlytalias/Bible-Database/edd4eb0a80ddeaea54ec0b2ff3e1cb72c09b85d0/English/bible.json"
const importBatchSize = 1000 // Insert verses in batches

// Helper functions to parse chapter and verse numbers
func parseChapterNumber(chapterStr string) (int, error) {
	// Try to parse as integer
	var chapter int
	_, err := fmt.Sscanf(chapterStr, "%d", &chapter)
	if err != nil {
		return 0, err
	}
	return chapter, nil
}

func parseVerseNumber(verseIdStr string) (int, error) {
	// Try to parse the verse ID as an integer
	// The verse ID might be padded with zeros
	var verse int
	_, err := fmt.Sscanf(verseIdStr, "%d", &verse)
	if err != nil {
		return 0, err
	}
	return verse + 1, nil // Add 1 since the IDs appear to be 0-based
}

func downloadAndImportKJV(ctx context.Context, collection *mongo.Collection) error {
	log.Printf("INFO: Attempting to download Bible data from %s", bibleJsonURL)
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Minute) // Timeout for download
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", bibleJsonURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download Bible JSON: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Bible JSON, status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Println("INFO: Bible data downloaded successfully. Parsing...")

	var bibleData BibleJSON
	if err := json.Unmarshal(bodyBytes, &bibleData); err != nil {
		return fmt.Errorf("failed to unmarshal Bible JSON: %w", err)
	}

	log.Printf("INFO: Parsed Bible data with %d books. Converting and preparing for import...", len(bibleData.Book))

	var versesToImport []interface{}
	bookIndexMap := make(map[string]int) // To assign consistent book indices
	nextBookIndex := 0
	totalVerses := 0

	for bookIdx, book := range bibleData.Book {
		bookName := fmt.Sprintf("Book %d", bookIdx+1) // Default name if not provided
		if book.Name != "" {
			bookName = book.Name
		}

		bookIndex, exists := bookIndexMap[bookName]
		if !exists {
			bookIndex = nextBookIndex
			bookIndexMap[bookName] = bookIndex
			nextBookIndex++
		}

		for chapterIdx, chapter := range book.Chapter {
			chapterNum := chapterIdx + 1 // Default to 1-based index
			if chapter.Chapter != "" {
				// Try to parse chapter number if provided
				if parsedChapter, err := parseChapterNumber(chapter.Chapter); err == nil {
					chapterNum = parsedChapter
				}
			}

			for verseIdx, verseData := range chapter.Verse {
				verseNum := verseIdx + 1 // Default to 1-based index
				if verseData.Verseid != "" {
					// Try to parse verse number if provided
					if parsedVerse, err := parseVerseNumber(verseData.Verseid); err == nil {
						verseNum = parsedVerse
					}
				}

				verse := BibleVerse{
					Book:        bookName,
					BookIndex:   bookIndex,
					Chapter:     chapterNum,
					Verse:       verseNum,
					Text:        verseData.Verse,
					Translation: "kjv",
				}
				totalVerses++
				versesToImport = append(versesToImport, verse)

				// Insert in batches
				if len(versesToImport) >= importBatchSize {
					log.Printf("INFO: Importing batch (%d verses)...", len(versesToImport))
					_, insertErr := collection.InsertMany(ctx, versesToImport)
					if insertErr != nil {
						return fmt.Errorf("failed to insert verse batch: %w", insertErr)
					}
					versesToImport = []interface{}{} // Reset batch
				}
			}
		}
	}

	// Insert any remaining verses
	if len(versesToImport) > 0 {
		log.Printf("INFO: Importing final batch (%d verses)...", len(versesToImport))
		_, insertErr := collection.InsertMany(ctx, versesToImport)
		if insertErr != nil {
			return fmt.Errorf("failed to insert final verse batch: %w", insertErr)
		}
	}

	log.Printf("INFO: Successfully imported %d total Bible verses.", totalVerses)

	// --- Create Indexes ---
	log.Println("INFO: Creating indexes for bible_verses collection...")
	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "book", Value: 1}, {Key: "chapter", Value: 1}, {Key: "verse", Value: 1}}},       // Compound index for specific lookups
		{Keys: bson.D{{Key: "book_index", Value: 1}, {Key: "chapter", Value: 1}, {Key: "verse", Value: 1}}}, // Index for ordering
		{Keys: bson.D{{Key: "text", Value: "text"}}},                                                        // Text index for searching
	}
	_, indexErr := collection.Indexes().CreateMany(ctx, indexModels)
	if indexErr != nil {
		// Log warning, but don't fail startup completely if index creation fails
		log.Printf("WARN: Failed to create indexes for bible_verses: %v", indexErr)
	} else {
		log.Println("INFO: Successfully created indexes for bible_verses.")
	}

	return nil
}

func main() {
	// Load Configuration
	cfg := config.Load()

	// --- Dependency Injection ---

	// 1. Set up MongoDB connection
	// Use a longer timeout to accommodate potential download/import
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
	// Get verse collection and check/import data
	versesCollection := mongoDB.Collection("bible_verses")
	verseCount, err := versesCollection.CountDocuments(ctx, bson.M{})

	if err != nil {
		// Log error but proceed assuming we might need to import or it might recover
		log.Printf("WARN: Error checking initial Bible verses count: %v. Will attempt to check/import data.", err)
		// Reset count to 0 to trigger import check if error occurred
		verseCount = 0
	}

	if verseCount == 0 {
		log.Println("INFO: No Bible verses found or initial check failed. Attempting download and import...")
		if importErr := downloadAndImportKJV(ctx, versesCollection); importErr != nil {
			// If import fails, log fatal. The app likely can't function without verses.
			log.Fatalf("FATAL: Failed to download and import Bible data: %v", importErr)
		}

		// Verify count after import attempt
		verseCount, err = versesCollection.CountDocuments(ctx, bson.M{}) // Re-check count
		if err != nil {
			log.Printf("WARN: Error verifying verse count after import: %v", err)
		}
	}

	// Always use the MongoDB repository now
	log.Printf("INFO: Using MongoDB repository for Bible verses (current count: %d).", verseCount)
	verseRepo := repository.NewMongoVerseRepository(mongoDB)

	// Initialize chat usage repository for rate limiting (in-memory)
	chatUsageRepo := repository.NewMemoryChatUsageRepository()
	log.Printf("INFO: Rate limiting configured: enabled=%v, limit=%d per day (in-memory storage)",
		cfg.ChatRateLimitEnabled, cfg.ChatRateLimitPerDay)

	// Create all services
	verseService := service.NewVerseService(verseRepo)
	chatService := service.NewChatService(openRouterClient, cfg.LLMModelName, verseService, chatUsageRepo, cfg)
	planService := service.NewPlanService(planRepo, openRouterClient, planningModelName)

	// Start weekly Bible plan generation scheduler
	service.StartWeeklyPlanScheduler(planService, cfg)

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
