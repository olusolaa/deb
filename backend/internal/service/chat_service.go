package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync" // Import sync for mutex

	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/llm"
	// repository package might not be needed here unless resetting interacts with it
)

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
	GetResponse(ctx context.Context, verse domain.DailyVerse, question string) (string, error)
	// New method to reset history
	ResetChatHistory(ctx context.Context) error
}

// --- Chat Service Implementation Update ---
type chatService struct {
	llmClient llm.LLMClient
	modelName string
	// No need to store repo here unless needed for chat
}

// NewChatService remains the same constructor
func NewChatService(client llm.LLMClient, modelName string) ChatService {
	return &chatService{
		llmClient: client,
		modelName: modelName,
	}
}

// GetResponse now manages history
func (s *chatService) GetResponse(ctx context.Context, verse domain.DailyVerse, question string) (string, error) {
	if question == "" {
		return "", errors.New("question cannot be empty")
	}

	// Lock for reading and potentially writing history
	conversationHistoryMutex.Lock()
	defer conversationHistoryMutex.Unlock()

	// Retrieve current history for the niece
	history, ok := conversationHistory[nieceConversationKey]
	if !ok {
		history = make([]llm.Message, 0)
		// Initialize history with system prompt and verse context if needed
		// For follow-ups, the verse context might already be in history.
		// Let's add the verse context on the *first* message of a *new* conversation.
		if verse.Text != "" && verse.Reference != "" {
			// Add verse context as the *first* user message in a new chat
			initialContext := fmt.Sprintf("Let's talk about the verse: %s\n\"%s\"\n\nMy first question is: %s",
				verse.Reference, verse.Text, question)
			history = append(history, llm.Message{Role: "user", Content: initialContext})
			// Log that we added initial context
			log.Printf("INFO: Initializing chat history for key '%s' with verse context.", nieceConversationKey)
		} else {
			// If verse is empty, just add the question
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
