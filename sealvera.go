// Package sealvera provides AI decision audit logging for Go applications.
// It captures, logs, and explains every decision your AI agents make.
//
// Quick start:
//
//	import "github.com/sealvera/sealvera-go"
//
//	func main() {
//	    sealvera.Init(sealvera.Config{
//	        Endpoint: "http://localhost:3000",
//	        APIKey:   "sv_...",
//	        Agent:    "my-agent",
//	    })
//
//	    result, err := sealvera.Wrap(ctx, sealvera.WrapOptions{
//	        Agent:  "payment-agent",
//	        Action: "approve_payment",
//	        Input:  payment,
//	    }, func() (any, error) {
//	        return processPayment(payment)
//	    })
//	}
package sealvera

import (
	"context"
	"fmt"
	"strings"
)

// Version is the current SDK version.
const Version = "0.1.0"

// Config holds the SealVera SDK configuration.
type Config struct {
	// Endpoint is the SealVera server URL (e.g. "http://localhost:3000")
	Endpoint string
	// APIKey is the API key from the server startup (starts with "sv_")
	APIKey string
	// Agent is the default agent name for all logged calls
	Agent string
	// Debug enables debug logging to stdout
	Debug bool
}

// WrapOptions configures a single wrapped agent call.
type WrapOptions struct {
	// Agent overrides the default agent name from Config
	Agent string
	// Action is a short label for what the agent is doing
	Action string
	// Input is the input data to log (will be JSON-serialized)
	Input any
}

// LogEntry is the structured log entry sent to the SealVera server.
type LogEntry struct {
	ID             string `json:"id"`
	Timestamp      string `json:"timestamp"`
	Agent          string `json:"agent"`
	Action         string `json:"action"`
	Decision       string `json:"decision"`
	Input          any    `json:"input"`
	Output         any    `json:"output"`
	Reasoning      string `json:"reasoning"`
	ReasoningSteps any    `json:"reasoning_steps,omitempty"`
	ModelUsed      string `json:"model,omitempty"`      // top-level model field — server reads entry.model
	RawContext     any    `json:"raw_context"`
	Provider       string `json:"provider,omitempty"`
}

var globalClient *Client

// Init initializes the SealVera SDK with the given configuration.
// This must be called before Wrap() or any other SDK functions.
//
//	sealvera.Init(sealvera.Config{
//	    Endpoint: "http://localhost:3000",
//	    APIKey:   "sv_your_key_here",
//	    Agent:    "my-agent",
//	})
func Init(cfg Config) error {
	if cfg.Endpoint == "" {
		return fmt.Errorf("sealvera: Endpoint is required")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("sealvera: APIKey is required")
	}
	if cfg.Agent == "" {
		cfg.Agent = "ai-agent"
	}

	globalClient = NewClient(cfg)

	if cfg.Debug {
		fmt.Printf("[SealVera] Initialized: endpoint=%s, agent=%s\n", cfg.Endpoint, cfg.Agent)
	}
	return nil
}

// Wrap runs fn and logs the result to the SealVera server.
// It captures the input, output, and inferred decision automatically.
//
//	result, err := sealvera.Wrap(ctx, sealvera.WrapOptions{
//	    Agent:  "payment-agent",
//	    Action: "approve_payment",
//	    Input:  payment,
//	}, func() (any, error) {
//	    return processPayment(payment)
//	})
func Wrap(ctx context.Context, opts WrapOptions, fn func() (any, error)) (any, error) {
	if globalClient == nil {
		return nil, fmt.Errorf("sealvera: not initialized — call sealvera.Init() first")
	}
	return globalClient.Wrap(ctx, opts, fn)
}

// SendLog sends a log entry directly to the SealVera server.
// Prefer using Wrap() for most use cases.
func SendLog(ctx context.Context, entry LogEntry) error {
	if globalClient == nil {
		return fmt.Errorf("sealvera: not initialized — call sealvera.Init() first")
	}
	return globalClient.SendLog(ctx, entry)
}

// inferDecision extracts a decision label from output text.
func inferDecision(text string) string {
	upper := strings.ToUpper(text)
	switch {
	case strings.Contains(upper, "APPROVED") || strings.Contains(upper, "ALLOWED"):
		return "APPROVED"
	case strings.Contains(upper, "REJECTED") || strings.Contains(upper, "DENIED"):
		return "REJECTED"
	case strings.Contains(upper, "FLAGGED"):
		return "FLAGGED"
	default:
		return "completed"
	}
}
