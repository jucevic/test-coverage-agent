package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	ClaudeAPIURL     = "https://api.anthropic.com/v1/messages"
	DefaultModel     = "claude-sonnet-4-5-20250929"
	MaxTokens        = 8000
	RetryMaxAttempts = 3
	RetryBaseDelay   = 2 * time.Second
)

// Client handles communication with Claude API
type Client struct {
	apiKey     string
	httpClient *http.Client
	model      string
}

// NewClient creates a new Claude API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		model: DefaultModel,
	}
}

// Message represents a Claude API message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents a Claude API request
type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// Response represents a Claude API response
type Response struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	ResetTime time.Time
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, resets at %v", e.ResetTime)
}

// SendMessage sends a message to Claude and returns the response
func (c *Client) SendMessage(prompt string) (string, error) {
	req := Request{
		Model:     c.model,
		MaxTokens: MaxTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	var lastErr error
	for attempt := 0; attempt < RetryMaxAttempts; attempt++ {
		if attempt > 0 {
			delay := RetryBaseDelay * time.Duration(1<<uint(attempt-1))
			time.Sleep(delay)
		}

		response, err := c.makeRequest(req)
		if err != nil {
			// Check if it's a rate limit error
			if rateLimitErr, ok := err.(*RateLimitError); ok {
				return "", rateLimitErr // Don't retry rate limits, let caller handle
			}

			lastErr = err
			continue
		}

		// Extract text from response
		if len(response.Content) > 0 {
			return response.Content[0].Text, nil
		}

		return "", fmt.Errorf("empty response from Claude API")
	}

	return "", fmt.Errorf("failed after %d attempts: %w", RetryMaxAttempts, lastErr)
}

// makeRequest performs the actual HTTP request
func (c *Client) makeRequest(req Request) (*Response, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", ClaudeAPIURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle rate limiting (429)
	if resp.StatusCode == http.StatusTooManyRequests {
		var resetTime time.Time
		var retryAfter int

		// Try to parse retry-after header
		if retryHeader := resp.Header.Get("retry-after"); retryHeader != "" {
			fmt.Sscanf(retryHeader, "%d", &retryAfter)
			resetTime = time.Now().Add(time.Duration(retryAfter) * time.Second)
		} else {
			// Default to 60 seconds if no header
			retryAfter = 60
			resetTime = time.Now().Add(60 * time.Second)
		}

		return nil, &RateLimitError{
			ResetTime:  resetTime,
			RetryAfter: retryAfter,
		}
	}

	// Handle other errors
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil {
			return nil, fmt.Errorf("API error: %s - %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse successful response
	var response Response
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}
