package sealvera

// agent.go — Per-agent wrapper for Go SDK
//
// Go can't detect SDK types at runtime or monkey-patch like JS/Python.
// Instead, NewAgent() creates a named agent handle. Call its Wrap method
// with your LLM call — the provider is inferred from context or specified.
//
// This is the Go-idiomatic equivalent of JS/Python createClient():
//
//   // JS/Python:
//   const agent = SealVera.createClient(openaiClient, { agent: "fraud-screener" })
//   await agent.chat.completions.create(...)
//
//   // Go equivalent:
//   agent := sealvera.NewAgent("fraud-screener")
//   result, err := agent.WrapOpenAI(ctx, "evaluate", input, func() (any, error) {
//       return openaiClient.Chat.Completions.New(ctx, params)
//   })

import "context"

// Agent is a named agent handle. Create one per agent in your application.
// All calls through an Agent are logged under its name automatically.
//
//	fraudAgent := sealvera.NewAgent("fraud-screener")
//	uwAgent    := sealvera.NewAgent("loan-underwriter")
type Agent struct {
	name string
}

// NewAgent creates a new named agent handle.
// This is the idiomatic Go entry point — create one Agent per logical agent
// in your application, then call its Wrap methods for each LLM call.
//
//	fraudAgent := sealvera.NewAgent("fraud-screener")
//	result, err := fraudAgent.WrapOpenAI(ctx, "screen", input, func() (any, error) {
//	    return openaiClient.Chat.Completions.New(ctx, params)
//	})
func NewAgent(name string) *Agent {
	return &Agent{name: name}
}

// WrapOpenAI wraps an OpenAI call and logs it under this agent's name.
//
//	result, err := agent.WrapOpenAI(ctx, "evaluate_application", input, func() (any, error) {
//	    return openaiClient.Chat.Completions.New(ctx, params)
//	})
func (a *Agent) WrapOpenAI(ctx context.Context, action string, input any, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "openai", ProviderOpts{Agent: a.name, Action: action, Input: input}, fn)
}

// WrapAnthropic wraps an Anthropic call and logs it under this agent's name.
// If your params include extended thinking, pass the response through as-is —
// SealVera extracts thinking blocks from the raw JSON automatically.
//
//	result, err := agent.WrapAnthropic(ctx, "evaluate", input, func() (any, error) {
//	    return anthropicClient.Messages.New(ctx, params)
//	})
func (a *Agent) WrapAnthropic(ctx context.Context, action string, input any, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "anthropic", ProviderOpts{Agent: a.name, Action: action, Input: input}, fn)
}

// WrapOpenRouter wraps an OpenRouter call and logs it under this agent's name.
// response.model is logged per entry so you know which model made each decision.
//
//	result, err := agent.WrapOpenRouter(ctx, "route", input, func() (any, error) {
//	    return openrouterClient.Chat.Completions.New(ctx, params)
//	})
func (a *Agent) WrapOpenRouter(ctx context.Context, action string, input any, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "openrouter", ProviderOpts{Agent: a.name, Action: action, Input: input}, fn)
}

// Wrap is the universal method — use when the provider doesn't matter
// or when you're using a custom/unsupported LLM.
func (a *Agent) Wrap(ctx context.Context, action string, input any, fn func() (any, error)) (any, error) {
	return wrapProvider(ctx, "llm", ProviderOpts{Agent: a.name, Action: action, Input: input}, fn)
}

// Name returns the agent's name.
func (a *Agent) Name() string { return a.name }
