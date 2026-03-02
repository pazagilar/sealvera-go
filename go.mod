module github.com/sealvera/sealvera-go

go 1.21

// No external dependencies — stdlib only.
// openai-go, anthropic-go, etc. are used by the caller, not the SDK.
// This keeps the integration completely dependency-free.
