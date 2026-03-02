package sealvera_test

import (
	"context"
	"testing"

	sealvera "github.com/sealvera/sealvera-go"
)

// TestNew verifies that NewClient can be created with a valid config.
func TestNew(t *testing.T) {
	cfg := sealvera.Config{
		Endpoint: "http://localhost:3000",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
		Agent:    "test-agent",
		Debug:    false,
	}

	client := sealvera.NewClient(cfg)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

// TestInit verifies that Init sets up the global client.
func TestInit(t *testing.T) {
	err := sealvera.Init(sealvera.Config{
		Endpoint: "http://localhost:3000",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
		Agent:    "test-agent",
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

// TestInitMissingEndpoint verifies that Init rejects missing endpoint.
func TestInitMissingEndpoint(t *testing.T) {
	err := sealvera.Init(sealvera.Config{
		APIKey: "sv_test_key",
		Agent:  "test-agent",
	})
	if err == nil {
		t.Fatal("expected error for missing Endpoint, got nil")
	}
}

// TestInitMissingAPIKey verifies that Init rejects missing API key.
func TestInitMissingAPIKey(t *testing.T) {
	err := sealvera.Init(sealvera.Config{
		Endpoint: "http://localhost:3000",
		Agent:    "test-agent",
	})
	if err == nil {
		t.Fatal("expected error for missing APIKey, got nil")
	}
}

// TestWrapLLM verifies WrapLLM runs the fn and returns its result.
// The log send will fail (no real server) but that's non-fatal (async).
func TestWrapLLM(t *testing.T) {
	// Re-init for this test
	sealvera.Init(sealvera.Config{ //nolint:errcheck
		Endpoint: "http://localhost:19999", // intentionally unreachable
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
		Agent:    "test-agent",
	})

	ctx := context.Background()

	result, err := sealvera.WrapLLM(ctx, "openai", sealvera.ProviderOpts{
		Agent:  "test-agent",
		Action: "classify",
		Input:  map[string]string{"text": "hello world"},
	}, func() (any, error) {
		// Simulated LLM response
		return map[string]any{
			"decision":  "APPROVED",
			"reasoning": "Test passed",
		}, nil
	})

	if err != nil {
		t.Fatalf("WrapLLM returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("WrapLLM returned nil result")
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["decision"] != "APPROVED" {
		t.Errorf("expected decision=APPROVED, got %v", m["decision"])
	}
}

// TestWrapOpenAI verifies WrapOpenAI wrapper.
func TestWrapOpenAI(t *testing.T) {
	sealvera.Init(sealvera.Config{ //nolint:errcheck
		Endpoint: "http://localhost:19999",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
		Agent:    "test-agent",
	})

	ctx := context.Background()
	result, err := sealvera.WrapOpenAI(ctx, sealvera.ProviderOpts{
		Agent:  "test-agent",
		Action: "generate",
		Input:  "test input",
	}, func() (any, error) {
		return map[string]any{"decision": "completed", "output": "hello"}, nil
	})

	if err != nil {
		t.Fatalf("WrapOpenAI error: %v", err)
	}
	if result == nil {
		t.Fatal("WrapOpenAI returned nil")
	}
}

// TestWrapAnthropic verifies WrapAnthropic wrapper.
func TestWrapAnthropic(t *testing.T) {
	sealvera.Init(sealvera.Config{ //nolint:errcheck
		Endpoint: "http://localhost:19999",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
		Agent:    "test-agent",
	})

	ctx := context.Background()
	result, err := sealvera.WrapAnthropic(ctx, sealvera.ProviderOpts{
		Agent:  "test-agent",
		Action: "analyze",
		Input:  "test input",
	}, func() (any, error) {
		return map[string]any{"decision": "FLAGGED", "reasoning": "needs review"}, nil
	})

	if err != nil {
		t.Fatalf("WrapAnthropic error: %v", err)
	}
	if result == nil {
		t.Fatal("WrapAnthropic returned nil")
	}
}

// TestClientWrap verifies that the client Wrap method works directly.
func TestClientWrap(t *testing.T) {
	client := sealvera.NewClient(sealvera.Config{
		Endpoint: "http://localhost:19999",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
		Agent:    "wrap-test-agent",
	})

	ctx := context.Background()
	result, err := client.Wrap(ctx, sealvera.WrapOptions{
		Agent:  "wrap-test-agent",
		Action: "process_payment",
		Input:  map[string]float64{"amount": 99.99},
	}, func() (any, error) {
		return map[string]any{
			"approved":  true,
			"reasoning": "Within limit",
		}, nil
	})

	if err != nil {
		t.Fatalf("Wrap error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["approved"] != true {
		t.Errorf("expected approved=true, got %v", m["approved"])
	}
}

// TestMarshalEntry verifies JSON serialization of log entries.
func TestMarshalEntry(t *testing.T) {
	entry := sealvera.LogEntry{
		ID:        "test-id-123",
		Timestamp: "2026-02-26T00:00:00Z",
		Agent:     "test-agent",
		Action:    "approve_payment",
		Decision:  "APPROVED",
		Input:     map[string]string{"amount": "100"},
		Output:    map[string]string{"status": "ok"},
		Reasoning: "Amount within threshold",
		RawContext: map[string]any{
			"agent":  "test-agent",
			"action": "approve_payment",
		},
	}

	b, err := sealvera.MarshalEntry(entry)
	if err != nil {
		t.Fatalf("MarshalEntry failed: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("MarshalEntry returned empty bytes")
	}
}
