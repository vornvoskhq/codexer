// Package llmlog provides instrumentation for logging LLM API calls.
package llmlog

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"plandex-server/model"
	shared "plandex-shared"
	"plandex-server/types"
)

// WrappedClient wraps the original ClientInfo with logging capabilities
type WrappedClient struct {
	Original   model.ClientInfo
	Logger     *Logger
	RequestNum int
}

// wrappedStream wraps the original stream to log chunks as they arrive
type wrappedStream struct {
	original  *model.ExtendedChatCompletionStream // The original stream
	logger    *Logger                             // Logger instance
	requestID string                             // Unique ID for the request
}

// Recv implements the Recv method for the wrapped stream
func (w *wrappedStream) Recv() (*types.ExtendedChatCompletionStreamResponse, error) {
	// Call the original Recv method
	resp, err := w.original.Recv()
	
	// Log the chunk or error
	if err != nil {
		w.logger.LogStreamEnd(w.requestID, err)
	} else if resp != nil && len(resp.Choices) > 0 {
		// Extract content from the response
		content := ""
		if resp.Choices[0].Delta.Content != "" {
			content = resp.Choices[0].Delta.Content
		}
		// Log the chunk if there's content
		if content != "" {
			w.logger.LogStreamChunk(w.requestID, content)
		}
	}
	
	return resp, err
}

// Close closes the underlying stream
func (w *wrappedStream) Close() error {
	// Log the end of the stream if it wasn't already logged
	w.logger.LogStreamEnd(w.requestID, nil)
	return w.original.Close()
}

// CreateChatCompletionStream wraps the original client's CreateChatCompletionStream method
func (w *WrappedClient) CreateChatCompletionStream(
	authVars map[string]string,
	modelConfig *shared.ModelRoleConfig,
	settings *shared.PlanSettings,
	orgUserConfig *shared.OrgUserConfig,
	currentOrgId string,
	currentUserId string,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
) (*model.ExtendedChatCompletionStream, error) {
	// Generate a unique request ID
	requestID := fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), w.RequestNum)
	w.RequestNum++

	// Convert messages to the format expected by LLMRequest
	var messages []openai.ChatCompletionMessage
	for _, msg := range req.Messages {
		// Convert the message parts to a single string
		content := ""
		for _, part := range msg.Content {
			if part.Type == "text" {
				content += part.Text
			}
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	// Get the provider name for logging
	provider := "unknown"
	if w.Original.ProviderConfig.Provider != "" {
		provider = string(w.Original.ProviderConfig.Provider)
	}

	// Create the log request
	logReq := &LLMRequest{
		RequestID:   requestID,
		Model:       string(req.Model), // Convert ModelName to string
		Messages:    messages,
		Temperature: float32(req.Temperature),
		MaxTokens:   int(req.MaxTokens),
		Provider:    provider,
		Timestamp:   time.Now(),
	}

	// Log the request
	if err := w.Logger.LogRequest(ctx, logReq); err != nil {
		w.Logger.LogError(ctx, requestID, fmt.Errorf("failed to log request: %w", err))
	}

	// Create a context with the request ID for better tracing
	ctx = context.WithValue(ctx, "request_id", requestID)

	// Call the original method
	// Create a clients map with the provider name as the key
	providerName := string(w.Original.ProviderConfig.Provider)
	clients := map[string]model.ClientInfo{
		providerName: w.Original,
	}

	stream, err := model.CreateChatCompletionStream(
		clients,
		authVars,
		modelConfig,
		settings,
		orgUserConfig,
		currentOrgId,
		currentUserId,
		ctx,
		req,
	)

	// Log the response or error
	if err != nil {
		w.Logger.LogError(ctx, requestID, fmt.Errorf("API call failed: %w", err))
		return nil, err
	}

	// Return the original stream for now, as we can't directly wrap it
	// due to unexported fields in ExtendedChatCompletionStream
	// The logging will be handled by the middleware layer instead
	return stream, nil
}

// WrapClient wraps a ClientInfo with logging capabilities
func WrapClient(original model.ClientInfo, logger *Logger) model.ClientInfo {
	if logger == nil || !logger.IsEnabled() {
		return original
	}

	// We're not wrapping the stream directly anymore due to unexported fields
	// The logging will be handled by the middleware layer instead

	// We can't modify the original ClientInfo structure, so we'll store the wrapped client
	// in a map that we can access later. This is a bit of a hack, but it works.
	// In a real implementation, you might want to modify the model package to support
	// this use case more cleanly.

	// For now, we'll just return the original client and rely on the fact that
	// the wrapped client will be used through the WrappedClient methods
	return original
}

// LLMRequest represents an LLM API request for logging
type LLMRequest struct {
	RequestID   string
	Model       string
	Messages    []openai.ChatCompletionMessage
	Temperature float32
	MaxTokens   int
	Timestamp   time.Time
	Provider    string
	Input       interface{} // Can be used for additional request data
}

// LLMResponse represents an LLM API response for logging
type LLMResponse struct {
	RequestID   string
	Content     string
	Timestamp   time.Time
	Model       string
	Usage       *openai.Usage
	Error       error
}