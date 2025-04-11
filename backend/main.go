package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	// Using standard library, but chi or gorilla/mux are good alternatives
	// "github.com/go-chi/chi/v5"
	// "github.com/go-chi/cors"
)

// --- Configuration ---
// IMPORTANT: Set your OpenAI API Key as an environment variable
// export OPENAI_API_KEY="your_actual_api_key_here"
var openAIAPIKey = os.Getenv("OPENAI_API_KEY")
var openAIAPIURL = "https://api.openai.com/v1/chat/completions" // Use the correct endpoint

// --- Data Structures ---

type BibleVerse struct {
	Book        string `json:"book"`
	Chapter     int    `json:"chapter"`
	VerseNumber int    `json:"verse"`
	Text        string `json:"text"`
	Reference   string `json:"reference"` // Combined reference like "John 3:16"
}

type ChatRequest struct {
	Verse    BibleVerse `json:"verse"`
	Question string     `json:"question"`
}

type ChatResponse struct {
	Answer string `json:"answer"`
}

// --- Verse Data ---
// A small, predefined list of verses. In a real app, this might come from a DB or external API.
// Using Day of Year (1-366) to pick a verse.
var dailyVerses = []BibleVerse{
	{Book: "John", Chapter: 3, VerseNumber: 16, Text: "For God so loved the world that he gave his one and only Son, that whoever believes in him shall not perish but have eternal life."},
	{Book: "Philippians", Chapter: 4, VerseNumber: 13, Text: "I can do all this through him who gives me strength."},
	{Book: "Jeremiah", Chapter: 29, VerseNumber: 11, Text: "For I know the plans I have for you,” declares the LORD, “plans to prosper you and not to harm you, plans to give you hope and a future."},
	{Book: "Romans", Chapter: 8, VerseNumber: 28, Text: "And we know that in all things God works for the good of those who love him, who have been called according to his purpose."},
	{Book: "Proverbs", Chapter: 3, VerseNumber: 5, Text: "Trust in the LORD with all your heart and lean not on your own understanding;"},
	{Book: "Psalm", Chapter: 23, VerseNumber: 1, Text: "The LORD is my shepherd, I lack nothing."},
	{Book: "Joshua", Chapter: 1, VerseNumber: 9, Text: "Have I not commanded you? Be strong and courageous. Do not be afraid; do not be discouraged, for the LORD your God will be with you wherever you go."},
	{Book: "Matthew", Chapter: 6, VerseNumber: 33, Text: "But seek first his kingdom and his righteousness, and all these things will be given to you as well."},
	{Book: "Isaiah", Chapter: 40, VerseNumber: 31, Text: "but those who hope in the LORD will renew their strength. They will soar on wings like eagles; they will run and not grow weary, they will walk and not be faint."},
	{Book: "1 Corinthians", Chapter: 13, VerseNumber: 4, Text: "Love is patient, love is kind. It does not envy, it does not boast, it is not proud."},
	// Add more verses here (ideally 366)
}

func getVerseForDay(dayOfYear int) BibleVerse {
	index := (dayOfYear - 1) % len(dailyVerses) // Loop back if dayOfYear > number of verses
	verse := dailyVerses[index]
	verse.Reference = fmt.Sprintf("%s %d:%d", verse.Book, verse.Chapter, verse.VerseNumber)
	return verse
}

// --- API Handlers ---

func verseHandler(w http.ResponseWriter, r *http.Request) {
	dayOfYear := time.Now().YearDay()
	verse := getVerseForDay(dayOfYear)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(verse)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	if openAIAPIKey == "" {
		http.Error(w, "OpenAI API key not configured on the server", http.StatusInternalServerError)
		log.Println("Error: OPENAI_API_KEY environment variable not set.")
		return
	}

	var chatReq ChatRequest
	err := json.NewDecoder(r.Body).Decode(&chatReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if chatReq.Question == "" || chatReq.Verse.Text == "" {
		http.Error(w, "Missing question or verse in request", http.StatusBadRequest)
		return
	}

	// Call LLM (OpenAI Example)
	answer, err := callOpenAI(chatReq.Verse, chatReq.Question)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get answer from LLM: %v", err), http.StatusInternalServerError)
		log.Printf("LLM API Error: %v", err)
		return
	}

	resp := ChatResponse{Answer: answer}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- LLM Interaction (OpenAI Example) ---

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float32         `json:"temperature,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func callOpenAI(verse BibleVerse, question string) (string, error) {
	// Construct the prompt for the LLM
	// Tailor the system prompt for a 14-year-old niece
	systemPrompt := "You are a friendly and knowledgeable assistant helping a teenager understand the Bible. Explain the verse clearly and simply, relating it to life where possible. Be encouraging and respectful."
	userPrompt := fmt.Sprintf("Regarding the Bible verse %s (%s): %s\n\nPlease answer this question: %s",
		verse.Reference, verse.Text,
		"\nHelp me understand this verse better.", // Adding context
		question)

	requestBody := OpenAIRequest{
		Model: "gpt-3.5-turbo", // Or "gpt-4" if you have access and prefer it
		Messages: []OpenAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   200, // Limit response length
		Temperature: 0.7, // Balance creativity and focus
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequest("POST", openAIAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openAIAPIKey)

	client := &http.Client{Timeout: 30 * time.Second} // Add a timeout
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		// Try reading the raw body for more context if JSON parsing fails
		bodyBytes := new(bytes.Buffer)
		bodyBytes.ReadFrom(resp.Body) // Read remaining body
		log.Printf("Raw OpenAI Response Body: %s", bodyBytes.String())
		return "", fmt.Errorf("failed to decode OpenAI response (status %d): %w. Raw body: %s", resp.StatusCode, err, bodyBytes.String())
	}

	// Check for API errors returned in the JSON response body
	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error (%s): %s", openAIResp.Error.Type, openAIResp.Error.Message)
	}

	// Check HTTP status code *after* attempting to decode potential JSON error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("OpenAI API returned non-success status: %s", resp.Status)
	}

	if len(openAIResp.Choices) > 0 {
		return strings.TrimSpace(openAIResp.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no response choices received from OpenAI")
}

// --- CORS Middleware (Simple Example) ---

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from your React app's origin (adjust if different)
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// --- Main Function ---

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/verse/today", verseHandler)
	mux.HandleFunc("/api/chat", chatHandler)

	// Wrap the mux with the CORS middleware
	handler := corsMiddleware(mux)

	fmt.Println("Backend server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
