package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenRouterClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewOpenRouterClient(apiKey, baseURL string) *OpenRouterClient {
	return &OpenRouterClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
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
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *OpenRouterClient) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	var response ChatCompletionResponse

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return response, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://publisher.app") // Required by OpenRouter

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return response, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, string(respBytes))
	}

	if err := json.Unmarshal(respBytes, &response); err != nil {
		return response, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}
