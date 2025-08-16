// Package llmlog: wrapper.go
// Provides a client wrapper to instrument LLM API calls for logging.
//
// The wrapper exposes PreCall and PostCall hooks for metadata capture.

package llmlog

import (
	"context"
	"time"
)

// LLMClient is a generic interface for LLM clients.
// This should match (or embed) the actual client interface used by Plandex.
type LLMClient interface {
	// CallLLM executes an LLM request.
	CallLLM(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	// TODO: Expand with additional methods as needed.
}

// LLMRequest and LLMResponse represent LLM API call data.
// These should be adapted to match actual Plandex LLM client types.
type LLMRequest struct {
	// TODO: Fill with fields (prompt, role, model, etc.)
}

type LLMResponse struct {
	// TODO: Fill with fields (response, tokens, error, etc.)
}

// PreCallMetadata holds information captured before the LLM call.
type PreCallMetadata struct {
	Timestamp   time.Time
	RequestID   string
	SessionID   string
	Role        string
	Provider    string
	Model       string
	Input       interface{} // TODO: Replace with concrete type.
	// TODO: Add other relevant fields.
}

// PostCallMetadata holds results and metrics after the LLM call.
type PostCallMetadata struct {
	Response   interface{} // TODO: Replace with concrete type.
	OutputTokens int
	Duration    time.Duration
	Status      string
	Error       error
	// TODO: Add other relevant fields as per roadmap.
}

// WrapClient returns an instrumented LLMClient that logs calls.
func WrapClient(original LLMClient, logger *Logger) LLMClient {
	// TODO: Return a struct implementing LLMClient, calling logger.Log() with PreCall/PostCall metadata.
	return &wrappedClient{
		inner:  original,
		logger: logger,
	}
}

// wrappedClient implements LLMClient and logs calls.
type wrappedClient struct {
	inner  LLMClient
	logger *Logger
}

func (w *wrappedClient) CallLLM(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// TODO: Capture PreCallMetadata, start timer
	// Call the real client
	// TODO: Capture PostCallMetadata, log result
	return w.inner.CallLLM(ctx, req)
}

// TODO: Add hooks for additional methods as needed.