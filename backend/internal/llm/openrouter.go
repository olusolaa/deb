package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// --- Interfaces ---

// LLMClient defines the interface for interacting with a language model.
type LLMClient interface {
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)
}

// --- OpenRouter Implementation ---

type OpenRouterClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Ensure OpenRouterClient implements LLMClient
var _ LLMClient = (*OpenRouterClient)(nil)

func NewOpenRouterClient(apiKey, baseURL string) *OpenRouterClient {
	return &OpenRouterClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Add a default timeout
		},
	}
}

// --- Structs (Copied from your example) ---

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model          string          `json:"model"` // e.g., "openai/gpt-3.5-turbo" or "google/gemini-pro" via OpenRouter
	Messages       []Message       `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"` // Add this field
	// Add other OpenRouter specific fields if needed (e.g., transforms, route)
}

type ResponseFormat struct {
	Type string `json:"type"` // e.g., "json_object"
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
	Usage *struct { // Use pointer for optional field
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	// Add Error field if OpenRouter returns errors within the JSON body
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"` // Can be string or int
	} `json:"error,omitempty"`
}

// --- Client Method (Copied and slightly adapted from your example) ---

func (c *OpenRouterClient) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	var response ChatCompletionResponse

	if c.apiKey == "" {
		return response, fmt.Errorf("OpenRouter API key is not configured")
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/chat/completions", c.baseURL) // OpenRouter usually uses /api/v1/
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return response, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	// IMPORTANT: OpenRouter requires these headers
	httpReq.Header.Set("HTTP-Referer", "urn:app://bible-app") // Replace with your app URL if deployed
	httpReq.Header.Set("X-Title", "Bible App Niece")          // Replace with your app name

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return response, fmt.Errorf("failed to send request to OpenRouter: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try unmarshalling first, as errors might be in the JSON body
	if err := json.Unmarshal(respBytes, &response); err != nil {
		// If unmarshalling fails, return the raw body for context, especially on non-200 status
		if httpResp.StatusCode != http.StatusOK {
			return response, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, string(respBytes))
		}
		// If status is OK but unmarshal failed, it's a different issue
		return response, fmt.Errorf("failed to unmarshal response (status %d): %w. Body: %s", httpResp.StatusCode, err, string(respBytes))
	}

	// Check for API errors *within* the JSON response
	if response.Error != nil {
		return response, fmt.Errorf("OpenRouter API error: type=%s, code=%v, message=%s", response.Error.Type, response.Error.Code, response.Error.Message)
	}

	// Check status code after attempting to parse potential JSON error messages
	if httpResp.StatusCode != http.StatusOK {
		// We already tried unmarshalling, so the error might be structured or just plain text
		errMsg := string(respBytes)
		if response.Error != nil { // If we *did* parse an error object
			errMsg = fmt.Sprintf("type=%s, code=%v, message=%s", response.Error.Type, response.Error.Code, response.Error.Message)
		}
		return response, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, errMsg)
	}

	return response, nil
}
