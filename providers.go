package sealvera

// providers.go — Multi-LLM support for SealVera Go SDK
//
// Since Go doesn't support monkey-patching like Python/JS, the Go SDK
// uses explicit wrapper functions for each LLM provider. Pass your LLM
// call as the fn argument and SealVera handles the logging.
//
// Usage patterns:
//
//   // OpenAI-compatible (openai-go, azure-openai-go, etc.)
//   result, err := sealvera.WrapOpenAI(ctx, sealvera.ProviderOpts{
//       Agent: "my-agent", Action: "decide", Input: data,
//   }, func() (*openai.ChatCompletion, error) {
//       return client.Chat.Completions.New(ctx, params)
//   })
//
//   // Anthropic
//   result, err := sealvera.WrapAnthropic(ctx, sealvera.ProviderOpts{...}, fn)
//
//   // Universal (any LLM)
//   result, err := sealvera.WrapLLM(ctx, "gemini", sealvera.ProviderOpts{...}, fn)

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ProviderOpts configures a provider-specific wrapped call.
type ProviderOpts struct {
	Agent  string // Agent name (overrides default)
	Action string // Action label
	Input  any    // Input data to log
}

// OpenAIResponse is a minimal interface for extracting content from OpenAI responses.
// Compatible with github.com/openai/openai-go and similar SDKs.
type OpenAIResponse interface {
	// We use JSON marshaling to extract content universally
}

// WrapOpenAI wraps an OpenAI-compatible call and logs it to SealVera.
// fn should return the raw API response (any type) and an error.
//
//	completion, err := sealvera.WrapOpenAI(ctx, sealvera.ProviderOpts{
//	    Agent: "my-agent", Action: "approve", Input: data,
//	}, func() (any, error) {
//	    return openaiClient.Chat.Completions.New(ctx, params)
//	})
func WrapOpenAI(ctx context.Context, opts ProviderOpts, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "openai", opts, fn)
}


// WrapAnthropic wraps an Anthropic Claude call and logs it to SealVera.
//
//	msg, err := sealvera.WrapAnthropic(ctx, sealvera.ProviderOpts{
//	    Agent: "my-agent", Action: "analyze", Input: data,
//	}, func() (any, error) {
//	    return anthropicClient.Messages.New(ctx, params)
//	})
func WrapAnthropic(ctx context.Context, opts ProviderOpts, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "anthropic", opts, fn)
}

// WrapGemini wraps a Google Gemini call and logs it to SealVera.
func WrapGemini(ctx context.Context, opts ProviderOpts, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "gemini", opts, fn)
}

// WrapOllama wraps an Ollama call and logs it to SealVera.
func WrapOllama(ctx context.Context, opts ProviderOpts, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "ollama", opts, fn)
}

// WrapLLM is the universal wrapper for any LLM provider.
// Use this for providers not covered by the specific wrappers above.
//
//	result, err := sealvera.WrapLLM(ctx, "cohere", sealvera.ProviderOpts{
//	    Agent: "my-agent", Action: "rerank", Input: data,
//	}, func() (any, error) {
//	    return cohereClient.Rerank(ctx, params)
//	})
func WrapLLM(ctx context.Context, provider string, opts ProviderOpts, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, provider, opts, fn)
}

// wrapProvider is the internal implementation for all provider wrappers.
func wrapProvider(ctx context.Context, provider string, opts ProviderOpts, fn func() (any, error)) (any, error) {
	if globalClient == nil {
		return nil, fmt.Errorf("sealvera: not initialized — call sealvera.Init() first")
	}

	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	agentName := opts.Agent
	if agentName == "" {
		agentName = globalClient.cfg.Agent
	}

	output, err := fn()

	decision := "completed"
	reasoning := ""
	modelUsed := ""

	if err != nil {
		decision = "error"
		output = map[string]string{"error": err.Error()}
	} else {
		// Marshal the response to a flat map so we can extract fields
		// regardless of whether output is a struct, map, or SDK type.
		b, jerr := json.Marshal(output)
		if jerr == nil {
			var m map[string]any
			if json.Unmarshal(b, &m) == nil {
				decision, reasoning, modelUsed = extractResponseFields(m, provider)
			} else {
				text := strings.Trim(string(b), `"`)
				decision = inferDecision(text)
			}
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
		ModelUsed: modelUsed,
		RawContext: map[string]any{
			"agent":    agentName,
			"action":   opts.Action,
			"input":    opts.Input,
			"output":   output,
			"provider": provider,
		},
		Provider: provider,
	}

	globalClient.sendLogAsync(entry)
	return output, err
}

// extractResponseFields pulls decision, reasoning, and model from a marshaled
// LLM response map. Handles:
//   - Plain maps (decision/model keys directly)
//   - OpenAI-go response structs (choices[0].message.content, model)
//   - Anthropic-go response structs (content[0].text, model)
//   - OpenRouter (same as OpenAI shape, model may be provider/model-name)
func extractResponseFields(m map[string]any, provider string) (decision, reasoning, modelUsed string) {
	decision = "completed"

	// ── Model — present at top level in all major SDK response types ──────
	if v, ok := m["model"].(string); ok && v != "" {
		modelUsed = v
	}

	// ── OpenAI / OpenRouter shape: choices[0].message.content ─────────────
	if choices, ok := m["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if msg, ok := choice["message"].(map[string]any); ok {
				content, _ := msg["content"].(string)
				if content != "" {
					decision = inferDecision(content)
					// Extract reasoning from structured JSON if present
					reasoning = extractReasoning(content)
				}
			}
		}
		return
	}

	// ── Anthropic shape: content[0].text ──────────────────────────────────
	if contents, ok := m["content"].([]any); ok && len(contents) > 0 {
		if block, ok := contents[0].(map[string]any); ok {
			// text block
			if text, ok := block["text"].(string); ok && text != "" {
				decision = inferDecision(text)
				reasoning = extractReasoning(text)
				return
			}
			// thinking block (extended thinking)
			if block["type"] == "thinking" {
				if thinkText, ok := block["thinking"].(string); ok {
					reasoning = thinkText
				}
				// decision may be in a subsequent text block
				if len(contents) > 1 {
					if b2, ok := contents[1].(map[string]any); ok {
						if text, ok := b2["text"].(string); ok && text != "" {
							decision = inferDecision(text)
						}
					}
				}
				return
			}
		}
	}

	// ── Plain map with a direct decision key (our own demo/test responses) ─
	if d, ok := m["decision"].(string); ok && d != "" {
		decision = strings.ToUpper(d)
	}
	if r, ok := m["reasoning"].(string); ok {
		reasoning = r
	}

	return
}

// extractReasoning pulls the "reasoning" string from a JSON response text,
// or returns empty if it's not structured JSON.
func extractReasoning(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		return ""
	}
	if r, ok := m["reasoning"].(string); ok {
		return r
	}
	if r, ok := m["reason"].(string); ok {
		return r
	}
	return ""
}
