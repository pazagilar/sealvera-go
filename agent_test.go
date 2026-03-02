package sealvera_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	sealvera "github.com/sealvera/sealvera-go"
)

// testEndpoint and testAPIKey — point at the live dev server.
// Tests use an intentionally unreachable port for unit-level checks,
// and the live server for integration-level log-verification checks.
const (
	testEndpoint = "http://localhost:3000"
	testAPIKey   = "ak_adc7a8c966cfc652d68df813c12b9970716fc67b093be07a"
)

// getRecentLog fetches the last 10 logs and finds the first for agentName.
func getRecentLog(t *testing.T, agentName string) map[string]any {
	t.Helper()
	url := fmt.Sprintf("%s/api/logs?limit=10", testEndpoint)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-SealVera-Key", testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("  [warn] getRecentLog HTTP error: %v", err)
		return nil
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Logf("  [warn] getRecentLog decode error: %v", err)
		return nil
	}
	logsRaw, ok := body["logs"]
	if !ok {
		return nil
	}
	logs, ok := logsRaw.([]any)
	if !ok {
		return nil
	}
	for _, l := range logs {
		entry, ok := l.(map[string]any)
		if !ok {
			continue
		}
		if entry["agent"] == agentName {
			return entry
		}
	}
	return nil
}

// ─── NewAgent construction ─────────────────────────────────────────────────

// TestNewAgent verifies that NewAgent returns a non-nil handle with the correct name.
func TestNewAgent(t *testing.T) {
	agent := sealvera.NewAgent("test-go-new-agent")
	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}
	if agent.Name() != "test-go-new-agent" {
		t.Errorf("expected name 'test-go-new-agent', got '%s'", agent.Name())
	}
}

// TestNewAgentMultiple verifies that separate agents are independent (different names).
func TestNewAgentMultiple(t *testing.T) {
	a1 := sealvera.NewAgent("fraud-screener")
	a2 := sealvera.NewAgent("loan-underwriter")
	if a1.Name() == a2.Name() {
		t.Error("expected different agent names, got the same")
	}
}

// ─── WrapOpenAI ────────────────────────────────────────────────────────────

// TestAgentWrapOpenAI_Returns verifies the fn result passes through correctly.
func TestAgentWrapOpenAI_Returns(t *testing.T) {
	if err := sealvera.Init(sealvera.Config{
		Endpoint: "http://localhost:19999", // unreachable — log send is async/non-fatal
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	agent := sealvera.NewAgent("test-go-openai-unit")
	ctx := context.Background()
	input := map[string]any{"applicant_id": "APP-GO-001", "amount": 25000}

	result, err := agent.WrapOpenAI(ctx, "evaluate_application", input, func() (any, error) {
		return map[string]any{
			"decision":   "APPROVED",
			"confidence": 0.94,
		}, nil
	})

	if err != nil {
		t.Fatalf("WrapOpenAI returned error: %v", err)
	}
	if result == nil {
		t.Fatal("WrapOpenAI returned nil result")
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["decision"] != "APPROVED" {
		t.Errorf("expected decision=APPROVED, got %v", m["decision"])
	}
}

// TestAgentWrapOpenAI_PropagatesError verifies that fn errors bubble up correctly.
func TestAgentWrapOpenAI_PropagatesError(t *testing.T) {
	sealvera.Init(sealvera.Config{ //nolint:errcheck
		Endpoint: "http://localhost:19999",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
	})

	agent := sealvera.NewAgent("test-go-openai-err")
	ctx := context.Background()

	_, err := agent.WrapOpenAI(ctx, "fail_action", nil, func() (any, error) {
		return nil, fmt.Errorf("simulated LLM timeout")
	})

	if err == nil {
		t.Fatal("expected error from fn, got nil")
	}
	if err.Error() != "simulated LLM timeout" {
		t.Errorf("expected 'simulated LLM timeout', got: %v", err)
	}
}

// ─── WrapAnthropic ─────────────────────────────────────────────────────────

// TestAgentWrapAnthropic_Returns verifies the fn result passes through correctly.
func TestAgentWrapAnthropic_Returns(t *testing.T) {
	sealvera.Init(sealvera.Config{ //nolint:errcheck
		Endpoint: "http://localhost:19999",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
	})

	agent := sealvera.NewAgent("test-go-anthropic-unit")
	ctx := context.Background()
	input := map[string]any{"claim_id": "CLM-GO-001", "amount": 8000}

	result, err := agent.WrapAnthropic(ctx, "review_claim", input, func() (any, error) {
		return map[string]any{
			"decision": "DENIED",
			"reason":   "duplicate_claim",
		}, nil
	})

	if err != nil {
		t.Fatalf("WrapAnthropic returned error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["decision"] != "DENIED" {
		t.Errorf("expected decision=DENIED, got %v", m["decision"])
	}
}

// ─── WrapOpenRouter ────────────────────────────────────────────────────────

// TestAgentWrapOpenRouter_Returns verifies the fn result passes through correctly.
func TestAgentWrapOpenRouter_Returns(t *testing.T) {
	sealvera.Init(sealvera.Config{ //nolint:errcheck
		Endpoint: "http://localhost:19999",
		APIKey:   "sv_test_0000000000000000000000000000000000000000000000000000",
	})

	agent := sealvera.NewAgent("test-go-openrouter-unit")
	ctx := context.Background()
	input := map[string]any{"loan_id": "LN-GO-007", "risk_score": 0.61}

	result, err := agent.WrapOpenRouter(ctx, "route_decision", input, func() (any, error) {
		return map[string]any{
			"decision": "REVIEW",
			"model":    "anthropic/claude-3-5-sonnet",
		}, nil
	})

	if err != nil {
		t.Fatalf("WrapOpenRouter returned error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["decision"] != "REVIEW" {
		t.Errorf("expected decision=REVIEW, got %v", m["decision"])
	}
}

// ─── Integration: logs reach the live server ───────────────────────────────

// TestAgentWrapOpenAI_Integration calls the live dev server and verifies the log was written.
func TestAgentWrapOpenAI_Integration(t *testing.T) {
	if err := sealvera.Init(sealvera.Config{
		Endpoint: testEndpoint,
		APIKey:   testAPIKey,
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	agent := sealvera.NewAgent("test-go-openai-integ")
	ctx := context.Background()
	input := map[string]any{
		"applicant_id": "APP-GO-INTEG-001",
		"session_id":   "go-integ-session",
	}

	result, err := agent.WrapOpenAI(ctx, "screen_application", input, func() (any, error) {
		return map[string]any{"decision": "APPROVED", "confidence": 0.97}, nil
	})

	if err != nil {
		t.Fatalf("WrapOpenAI error: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	time.Sleep(500 * time.Millisecond)

	log := getRecentLog(t, "test-go-openai-integ")
	if log == nil {
		t.Error("no log found for test-go-openai-integ — check server is running")
		return
	}
	t.Logf("  log id: %v  |  decision: %v  |  model: %v", log["id"], log["decision"], log["model_used"])
	if log["agent"] != "test-go-openai-integ" {
		t.Errorf("unexpected agent: %v", log["agent"])
	}
}

// TestAgentWrapAnthropic_Integration calls the live dev server and verifies the log was written.
func TestAgentWrapAnthropic_Integration(t *testing.T) {
	if err := sealvera.Init(sealvera.Config{
		Endpoint: testEndpoint,
		APIKey:   testAPIKey,
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	agent := sealvera.NewAgent("test-go-anthropic-integ")
	ctx := context.Background()
	input := map[string]any{
		"claim_id":   "CLM-GO-INTEG-001",
		"session_id": "go-integ-session",
	}

	result, err := agent.WrapAnthropic(ctx, "review_claim", input, func() (any, error) {
		return map[string]any{"decision": "DENIED", "reason": "duplicate"}, nil
	})

	if err != nil {
		t.Fatalf("WrapAnthropic error: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	time.Sleep(500 * time.Millisecond)

	log := getRecentLog(t, "test-go-anthropic-integ")
	if log == nil {
		t.Error("no log found for test-go-anthropic-integ")
		return
	}
	t.Logf("  log id: %v  |  decision: %v  |  model: %v", log["id"], log["decision"], log["model_used"])
	if log["agent"] != "test-go-anthropic-integ" {
		t.Errorf("unexpected agent: %v", log["agent"])
	}
}

// TestAgentWrapOpenRouter_Integration calls the live dev server and verifies the log was written.
func TestAgentWrapOpenRouter_Integration(t *testing.T) {
	if err := sealvera.Init(sealvera.Config{
		Endpoint: testEndpoint,
		APIKey:   testAPIKey,
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	agent := sealvera.NewAgent("test-go-openrouter-integ")
	ctx := context.Background()
	input := map[string]any{
		"loan_id":    "LN-GO-INTEG-007",
		"session_id": "go-integ-session",
	}

	result, err := agent.WrapOpenRouter(ctx, "route_decision", input, func() (any, error) {
		return map[string]any{"decision": "REVIEW", "model": "anthropic/claude-3-5-sonnet"}, nil
	})

	if err != nil {
		t.Fatalf("WrapOpenRouter error: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	time.Sleep(500 * time.Millisecond)

	log := getRecentLog(t, "test-go-openrouter-integ")
	if log == nil {
		t.Error("no log found for test-go-openrouter-integ")
		return
	}
	t.Logf("  log id: %v  |  decision: %v  |  model: %v", log["id"], log["decision"], log["model_used"])
	if log["agent"] != "test-go-openrouter-integ" {
		t.Errorf("unexpected agent: %v", log["agent"])
	}
}
