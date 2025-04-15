package repository

import (
	"bibleapp/backend/internal/llm"
	"context"
	"fmt"
	"log"
)

// VerseRepository defines the interface for fetching verse content by reference
type VerseRepository interface {
	GetVerseByReference(ctx context.Context, reference string) (string, error)
}

// LLMVerseRepository uses an LLM to fetch verse content when needed
type LLMVerseRepository struct {
	llmClient llm.LLMClient
	modelName string
}

// NewLLMVerseRepository creates a new repository that uses LLM to fetch verse content
func NewLLMVerseRepository(llmClient llm.LLMClient, modelName string) VerseRepository {
	return &LLMVerseRepository{
		llmClient: llmClient,
		modelName: modelName,
	}
}

// GetVerseByReference fetches the full text of a Bible verse by its reference using LLM
func (r *LLMVerseRepository) GetVerseByReference(ctx context.Context, reference string) (string, error) {
	log.Printf("INFO: Fetching verse content for reference: %s", reference)

	// Create a simple prompt to fetch the verse text
	systemPrompt := `You are a helpful Bible assistant. Your task is to provide ONLY the exact text of Bible verses when given a reference. 

Respond ONLY with the verse text, with no additional commentary, explanation, or formatting. 
Do not include the verse reference in your response, just the plain text of the verse(s).`

	userPrompt := fmt.Sprintf("Please provide the exact text for %s", reference)

	// Call the LLM to get the verse text
	request := llm.ChatCompletionRequest{
		Model: r.modelName,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   500,
		Temperature: 0.0, // Use low temperature for consistent responses
	}

	response, err := r.llmClient.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to fetch verse content: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("received empty response when fetching verse content")
	}

	// Return the verse text
	return response.Choices[0].Message.Content, nil
}
