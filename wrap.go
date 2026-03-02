package sealvera

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Wrap runs fn, logs the result, and returns whatever fn returned.
// If fn returns an error, the error is returned immediately and the log
// entry records the error as the output.
//
// Example:
//
//	result, err := client.Wrap(ctx, sealvera.WrapOptions{
//	    Agent:  "payment-agent",
//	    Action: "approve_payment",
//	    Input:  payment,
//	}, func() (any, error) {
//	    return processPayment(ctx, payment)
//	})
func (c *Client) Wrap(ctx context.Context, opts WrapOptions, fn func() (any, error)) (any, error) {
	startedAt := time.Now().UTC().Format(time.RFC3339Nano)

	agentName := opts.Agent
	if agentName == "" {
		agentName = c.cfg.Agent
	}

	output, fnErr := fn()

	decision := "completed"
	reasoning := ""

	if fnErr != nil {
		decision = "error"
		output = map[string]string{"error": fnErr.Error()}
	} else {
		// Try to extract decision from the output
		switch v := output.(type) {
		case map[string]any:
			if d, ok := v["decision"].(string); ok && d != "" {
				decision = d
			} else if d, ok := v["action"].(string); ok && d != "" {
				decision = d
			} else if approved, ok := v["approved"].(bool); ok {
				if approved {
					decision = "APPROVED"
				} else {
					decision = "REJECTED"
				}
			}
			if r, ok := v["reasoning"].(string); ok {
				reasoning = r
			}
		case string:
			decision = inferDecision(v)
		}
	}

	entry := LogEntry{
		ID:        newUUID(),
		Timestamp: startedAt,
		Agent:     agentName,
		Action:    opts.Action,
		Decision:  decision,
		Input:     opts.Input,
		Output:    output,
		Reasoning: reasoning,
		RawContext: map[string]any{
			"agent":  agentName,
			"action": opts.Action,
			"input":  opts.Input,
			"output": output,
		},
	}

	// Send log asynchronously (non-blocking)
	c.sendLogAsync(entry)

	return output, fnErr
}

// newUUID generates a new UUID v4 using crypto/rand.
func newUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback: use timestamp
		return fmt.Sprintf("ak-%d", time.Now().UnixNano())
	}
	// Set version 4 bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

// MarshalEntry marshals a log entry to JSON bytes (for debugging/testing).
func MarshalEntry(entry LogEntry) ([]byte, error) {
	return json.Marshal(entry)
}

// MustMarshalEntry marshals a log entry to JSON, panicking on error.
func MustMarshalEntry(entry LogEntry) []byte {
	b, err := MarshalEntry(entry)
	if err != nil {
		panic(fmt.Sprintf("sealvera: failed to marshal entry: %v", err))
	}
	return b
}
