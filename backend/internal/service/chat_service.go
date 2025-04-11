package service

import (
	"context"
	"errors"
	"fmt"

	"bibleapp/backend/internal/domain"
	"bibleapp/backend/internal/llm"
)

// ChatService defines the interface for chat operations.
type ChatService interface {
	GetResponse(ctx context.Context, verse domain.BibleVerse, question string) (string, error)
}

type chatService struct {
	llmClient llm.LLMClient
	modelName string // e.g., "openai/gpt-3.5-turbo"
}

// NewChatService creates a new ChatService.
func NewChatService(client llm.LLMClient, modelName string) ChatService {
	return &chatService{
		llmClient: client,
		modelName: modelName,
	}
}

// GetResponse uses the LLM client to get an answer to a question about a verse.
func (s *chatService) GetResponse(ctx context.Context, verse domain.BibleVerse, question string) (string, error) {
	if question == "" {
		return "", errors.New("question cannot be empty")
	}
	if verse.Text == "" || verse.Reference == "" {
		return "", errors.New("verse text and reference cannot be empty")
	}

	// Construct the prompt for the LLM
	systemPrompt := "You are a friendly, kind, and knowledgeable Bible helper explaining things to a 14-year-old. Explain the verse clearly and simply. Keep answers concise and encouraging. Relate it to modern life if appropriate, but stay true to the verse's meaning."
	userPrompt := fmt.Sprintf("Here's the Bible verse: %s\n\"%s\"\n\nMy question is: %s",
		verse.Reference, verse.Text, question)

	request := llm.ChatCompletionRequest{
		Model: s.modelName, // Use the configured model
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   250, // Adjust as needed
		Temperature: 0.6, // Slightly less random for factual explanation
	}

	response, err := s.llmClient.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("LLM completion failed: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return "", errors.New("LLM returned an empty response")
	}

	return response.Choices[0].Message.Content, nil
}
