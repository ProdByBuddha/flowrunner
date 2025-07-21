package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LLMProvider represents the type of LLM provider
type LLMProvider string

const (
	// OpenAI provider
	OpenAI LLMProvider = "openai"
	// Anthropic provider
	Anthropic LLMProvider = "anthropic"
	// Generic provider for custom APIs
	Generic LLMProvider = "generic"
)

// LLMClient provides a unified interface for interacting with different LLM providers
type LLMClient struct {
	httpClient *HTTPClient
	provider   LLMProvider
	apiKey     string
	baseURL    string
	options    map[string]interface{}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMRequest represents a request to an LLM
type LLMRequest struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Functions   []FunctionDefinition   `json:"functions,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// FunctionDefinition represents a function that can be called by the LLM
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolDefinition represents a tool that can be used by the LLM
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// LLMResponse represents a response from an LLM
type LLMResponse struct {
	ID                string                 `json:"id,omitempty"`
	Object            string                 `json:"object,omitempty"`
	Created           int64                  `json:"created,omitempty"`
	Model             string                 `json:"model,omitempty"`
	Choices           []Choice               `json:"choices,omitempty"`
	Usage             Usage                  `json:"usage,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Error             *ErrorInfo             `json:"error,omitempty"`
	RawResponse       map[string]interface{} `json:"raw_response,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewLLMClient creates a new LLM client
func NewLLMClient(provider LLMProvider, apiKey string, options map[string]interface{}) *LLMClient {
	client := &LLMClient{
		httpClient: NewHTTPClient(),
		provider:   provider,
		apiKey:     apiKey,
		options:    options,
	}

	// Set base URL based on provider
	switch provider {
	case OpenAI:
		client.baseURL = "https://api.openai.com/v1"
	case Anthropic:
		client.baseURL = "https://api.anthropic.com/v1"
	case Generic:
		if baseURL, ok := options["base_url"].(string); ok {
			client.baseURL = baseURL
		}
	}

	return client
}

// Complete sends a completion request to the LLM
func (c *LLMClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	switch c.provider {
	case OpenAI:
		return c.completeOpenAI(ctx, request)
	case Anthropic:
		// For Claude 3 models, use the messages API
		if strings.Contains(request.Model, "claude-3") {
			return c.completeAnthropicMessages(ctx, request)
		}
		// For older Claude models, use the legacy API
		return c.completeAnthropic(ctx, request)
	case Generic:
		return c.completeGeneric(ctx, request)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.provider)
	}
}

// completeOpenAI sends a completion request to OpenAI
func (c *LLMClient) completeOpenAI(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	// Create request body
	requestBody := map[string]interface{}{
		"model":       request.Model,
		"messages":    request.Messages,
		"temperature": request.Temperature,
	}

	if request.MaxTokens > 0 {
		requestBody["max_tokens"] = request.MaxTokens
	}

	if len(request.Stop) > 0 {
		requestBody["stop"] = request.Stop
	}

	if request.Stream {
		requestBody["stream"] = request.Stream
	}

	if len(request.Functions) > 0 {
		requestBody["functions"] = request.Functions
	}

	if len(request.Tools) > 0 {
		requestBody["tools"] = request.Tools
	}

	// Add any additional options
	for key, value := range request.Options {
		requestBody[key] = value
	}

	// Create HTTP request
	httpRequest := &HTTPRequest{
		URL:    fmt.Sprintf("%s/chat/completions", c.baseURL),
		Method: "POST",
		Body:   requestBody,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", c.apiKey),
			"Content-Type":  "application/json",
		},
		Timeout: 60 * time.Second,
	}

	// Execute request
	resp, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(resp.RawBody, &errorResp); err != nil {
			return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(resp.RawBody))
		}
		return &LLMResponse{
			Error: &ErrorInfo{
				Message: errorResp.Error.Message,
				Type:    errorResp.Error.Type,
				Code:    errorResp.Error.Code,
			},
		}, nil
	}

	// Parse response
	var openAIResp LLMResponse
	if err := json.Unmarshal(resp.RawBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	// Store raw response
	if rawMap, ok := resp.Body.(map[string]interface{}); ok {
		openAIResp.RawResponse = rawMap
	}

	return &openAIResp, nil
}

// completeAnthropicMessages sends a completion request to Anthropic using the messages API (Claude 3)
func (c *LLMClient) completeAnthropicMessages(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	// Extract system message if present
	var systemPrompt string
	var userAssistantMessages []Message

	for _, msg := range request.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			userAssistantMessages = append(userAssistantMessages, msg)
		}
	}

	// Create request body for Claude 3 messages API
	requestBody := map[string]interface{}{
		"model":       request.Model,
		"messages":    userAssistantMessages,
		"temperature": request.Temperature,
	}

	// Add system prompt if present
	if systemPrompt != "" {
		requestBody["system"] = systemPrompt
	}

	if request.MaxTokens > 0 {
		requestBody["max_tokens"] = request.MaxTokens
	}

	if len(request.Stop) > 0 {
		requestBody["stop_sequences"] = request.Stop
	}

	// Add any additional options
	for key, value := range request.Options {
		requestBody[key] = value
	}

	// Create HTTP request
	httpRequest := &HTTPRequest{
		URL:    fmt.Sprintf("%s/messages", c.baseURL),
		Method: "POST",
		Body:   requestBody,
		Headers: map[string]string{
			"x-api-key":         c.apiKey,
			"anthropic-version": "2023-06-01",
			"Content-Type":      "application/json",
		},
		Timeout: 60 * time.Second,
	}

	// Execute request
	resp, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API request failed: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(resp.RawBody))
	}

	// Parse response
	var anthropicResp struct {
		ID           string                   `json:"id"`
		Type         string                   `json:"type"`
		Role         string                   `json:"role"`
		Content      []map[string]interface{} `json:"content"`
		Model        string                   `json:"model"`
		StopReason   string                   `json:"stop_reason"`
		StopSequence string                   `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(resp.RawBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	// Extract text content from the response
	var content string
	for _, block := range anthropicResp.Content {
		if blockType, ok := block["type"].(string); ok && blockType == "text" {
			if textContent, ok := block["text"].(string); ok {
				content = textContent
				break
			}
		}
	}

	// Convert to standard format
	llmResp := &LLMResponse{
		ID:    anthropicResp.ID,
		Model: anthropicResp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: anthropicResp.StopReason,
			},
		},
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	// Store raw response
	if rawMap, ok := resp.Body.(map[string]interface{}); ok {
		llmResp.RawResponse = rawMap
	}

	return llmResp, nil
}

// completeAnthropic sends a completion request to Anthropic using the legacy API
func (c *LLMClient) completeAnthropic(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	// Convert messages to Anthropic format
	var systemPrompt string
	var userMessages []string
	var assistantMessages []string

	for _, msg := range request.Messages {
		switch msg.Role {
		case "system":
			systemPrompt = msg.Content
		case "user":
			userMessages = append(userMessages, msg.Content)
		case "assistant":
			assistantMessages = append(assistantMessages, msg.Content)
		}
	}

	// Build the prompt in Anthropic's format
	var prompt string
	if systemPrompt != "" {
		prompt = systemPrompt + "\n\n"
	}

	// Interleave user and assistant messages
	for i := 0; i < len(userMessages); i++ {
		prompt += "Human: " + userMessages[i] + "\n\n"
		if i < len(assistantMessages) {
			prompt += "Assistant: " + assistantMessages[i] + "\n\n"
		}
	}

	// Add final "Assistant: " if we're expecting a response
	prompt += "Assistant: "

	// Create request body
	requestBody := map[string]interface{}{
		"model":       request.Model,
		"prompt":      prompt,
		"temperature": request.Temperature,
	}

	if request.MaxTokens > 0 {
		requestBody["max_tokens_to_sample"] = request.MaxTokens
	}

	if len(request.Stop) > 0 {
		requestBody["stop_sequences"] = request.Stop
	}

	// Add any additional options
	for key, value := range request.Options {
		requestBody[key] = value
	}

	// Create HTTP request
	httpRequest := &HTTPRequest{
		URL:    fmt.Sprintf("%s/complete", c.baseURL),
		Method: "POST",
		Body:   requestBody,
		Headers: map[string]string{
			"X-API-Key":         c.apiKey,
			"anthropic-version": "2023-06-01",
			"Content-Type":      "application/json",
		},
		Timeout: 60 * time.Second,
	}

	// Execute request
	resp, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API request failed: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(resp.RawBody))
	}

	// Parse response
	var anthropicResp struct {
		Completion string `json:"completion"`
		StopReason string `json:"stop_reason"`
		Model      string `json:"model"`
	}
	if err := json.Unmarshal(resp.RawBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	// Convert to standard format
	llmResp := &LLMResponse{
		Model: anthropicResp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: anthropicResp.Completion,
				},
				FinishReason: anthropicResp.StopReason,
			},
		},
	}

	// Store raw response
	if rawMap, ok := resp.Body.(map[string]interface{}); ok {
		llmResp.RawResponse = rawMap
	}

	return llmResp, nil
}

// completeGeneric sends a completion request to a generic API
func (c *LLMClient) completeGeneric(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	// Get endpoint from options
	endpoint, ok := c.options["endpoint"].(string)
	if !ok {
		endpoint = "/v1/chat/completions"
	}

	// Create HTTP request
	httpRequest := &HTTPRequest{
		URL:    fmt.Sprintf("%s%s", c.baseURL, endpoint),
		Method: "POST",
		Body:   request,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", c.apiKey),
			"Content-Type":  "application/json",
		},
		Timeout: 60 * time.Second,
	}

	// Execute request
	resp, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LLM API error (status %d): %s", resp.StatusCode, string(resp.RawBody))
	}

	// Parse response
	var llmResp LLMResponse
	if err := json.Unmarshal(resp.RawBody, &llmResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Store raw response
	if rawMap, ok := resp.Body.(map[string]interface{}); ok {
		llmResp.RawResponse = rawMap
	}

	return &llmResp, nil
}
