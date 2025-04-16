package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync" // Import sync for mutex

	"bibleapp/backend/internal/config"
	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/llm"
	"bibleapp/backend/internal/repository"
)

// ErrRateLimitExceeded is returned when a user has exceeded their daily chat limit
type ErrRateLimitExceeded struct{}

func (e ErrRateLimitExceeded) Error() string {
	return "rate limit exceeded"
}

// --- Conversation Store (In-Memory) ---
// For simplicity, we'll use a single hardcoded key for the niece's conversation.
// In a multi-user app, this key would be dynamic (e.g., user ID, session ID).
const nieceConversationKey = "niece_conversation"

var (
	// Protect concurrent access to the history map
	conversationHistory      = make(map[string][]llm.Message)
	conversationHistoryMutex sync.RWMutex
)

// --- Chat Service Interface Update ---
type ChatService interface {
	// GetResponse now implicitly uses stored history
	GetResponse(ctx context.Context, verse domain.DailyVerse, question string, userID string) (string, error)
	// New method to reset history
	ResetChatHistory(ctx context.Context) error
	// Get current chat usage for a user
	GetChatUsage(ctx context.Context, userID string) (int, int, error) // returns (current usage, limit, error)
}

// --- Chat Service Implementation Update ---
type chatService struct {
	llmClient       llm.LLMClient
	modelName       string
	verseService    VerseService // Added verse service for Bible verse lookups
	chatUsageRepo   repository.ChatUsageRepository
	rateLimitConfig *config.Config // Configuration for rate limiting
}

// NewChatService now includes all dependencies
func NewChatService(client llm.LLMClient, modelName string, verseService VerseService,
	chatUsageRepo repository.ChatUsageRepository, cfg *config.Config) ChatService {
	return &chatService{
		llmClient:       client,
		modelName:       modelName,
		verseService:    verseService,
		chatUsageRepo:   chatUsageRepo,
		rateLimitConfig: cfg,
	}
}

// GetResponse now manages history and implements rate limiting
func (s *chatService) GetResponse(ctx context.Context, verse domain.DailyVerse, question string, userID string) (string, error) {
	if question == "" {
		return "", errors.New("question cannot be empty")
	}

	// Check rate limits if enabled
	if s.rateLimitConfig.ChatRateLimitEnabled && userID != "" {
		currentUsage, err := s.chatUsageRepo.GetTodayUsage(ctx, userID)
		if err != nil {
			log.Printf("WARN: Failed to check chat rate limit: %v", err)
			// Continue despite error to maintain service availability
		} else if currentUsage >= s.rateLimitConfig.ChatRateLimitPerDay {
			// User has exceeded their daily limit
			return "", ErrRateLimitExceeded{}
		}
	}

	// Lock for reading and potentially writing history
	conversationHistoryMutex.Lock()
	defer conversationHistoryMutex.Unlock()

	// Retrieve current history for the niece
	history, ok := conversationHistory[nieceConversationKey]
	if !ok {
		history = make([]llm.Message, 0)
		// Initialize history with verse reference for a new conversation (without full text to save tokens)
		if verse.Reference != "" {
			// Add only the verse reference as context, not the full text
			initialContext := fmt.Sprintf("Let's talk about Bible verse %s. The user can see the full text. My first question is: %s",
				verse.Reference, question)
			history = append(history, llm.Message{Role: "user", Content: initialContext})
			log.Printf("INFO: Initializing chat history for key '%s' with verse reference only (token optimized).", nieceConversationKey)
		} else {
			// If verse reference is empty, just add the question
			history = append(history, llm.Message{Role: "user", Content: question})
		}
	} else {
		// Append only the new question if history exists
		history = append(history, llm.Message{Role: "user", Content: question})
	}

	// --- LLM Prompt Construction ---
	// System prompt provides overall context
	systemPrompt := "You are a friendly, kind, and knowledgeable Bible helper explaining things to a 14-year-old. Explain the verse clearly and simply. Keep answers concise and encouraging. Relate it to modern life if appropriate, but stay true to the verse's meaning. Respond directly to the user's latest question, considering the conversation history provided."

	// Prepare messages for LLM: System Prompt + History
	// WARNING: Long histories can exceed LLM token limits. Production apps need pruning.
	messagesForLLM := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}
	messagesForLLM = append(messagesForLLM, history...)

	// Optional: Implement history pruning here if needed
	// e.g., keep only the last N messages, or based on token count

	request := llm.ChatCompletionRequest{
		Model:       s.modelName,
		Messages:    messagesForLLM,
		MaxTokens:   300, // Might need adjustment based on expected answer length
		Temperature: 0.6,
	}

	response, err := s.llmClient.CreateChatCompletion(ctx, request)
	if err != nil {
		// Don't save history if LLM fails
		return "", fmt.Errorf("LLM completion failed: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		// Don't save history if LLM gives empty response
		return "", errors.New("LLM returned an empty response")
	}

	assistantResponse := response.Choices[0].Message.Content

	// Append assistant's response to the history
	history = append(history, llm.Message{Role: "assistant", Content: assistantResponse})

	// Save updated history back to the map
	conversationHistory[nieceConversationKey] = history
	log.Printf("INFO: Updated chat history for key '%s'. History length: %d messages.", nieceConversationKey, len(history))

	// Increment usage counter for rate limiting if enabled and we have a user ID
	if s.rateLimitConfig.ChatRateLimitEnabled && userID != "" {
		newCount, err := s.chatUsageRepo.IncrementUsage(ctx, userID)
		if err != nil {
			log.Printf("WARN: Failed to increment chat usage counter: %v", err)
		} else {
			log.Printf("INFO: User %s has used %d/%d chat requests today",
				userID, newCount, s.rateLimitConfig.ChatRateLimitPerDay)
		}
	}

	// Return only the latest assistant response
	return assistantResponse, nil
}

// ResetChatHistory clears the conversation history for the niece
func (s *chatService) ResetChatHistory(ctx context.Context) error {
	conversationHistoryMutex.Lock()
	defer conversationHistoryMutex.Unlock()

	_, exists := conversationHistory[nieceConversationKey]
	if exists {
		delete(conversationHistory, nieceConversationKey)
		log.Printf("INFO: Chat history reset for key '%s'.", nieceConversationKey)
	} else {
		log.Printf("INFO: Chat history reset requested, but no history found for key '%s'.", nieceConversationKey)
	}
	return nil // Indicate success even if no history existed
}

// GetChatUsage returns the current usage and limit for a user
func (s *chatService) GetChatUsage(ctx context.Context, userID string) (int, int, error) {
	if !s.rateLimitConfig.ChatRateLimitEnabled || userID == "" {
		// Rate limiting is disabled or no user ID provided
		return 0, s.rateLimitConfig.ChatRateLimitPerDay, nil
	}

	currentUsage, err := s.chatUsageRepo.GetTodayUsage(ctx, userID)
	if err != nil {
		return 0, s.rateLimitConfig.ChatRateLimitPerDay, err
	}

	return currentUsage, s.rateLimitConfig.ChatRateLimitPerDay, nil
}
