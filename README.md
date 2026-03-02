# SealVera Go SDK

**Tamper-evident audit trails for AI agents — compliance-ready in minutes.**

[![Go Reference](https://pkg.go.dev/badge/github.com/sealvera/sealvera-go.svg)](https://pkg.go.dev/github.com/sealvera/sealvera-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/go-%3E%3D1.21-brightgreen)](https://go.dev)

SealVera gives every AI decision a cryptographically-sealed, immutable audit log — so you can prove what your agent decided, why it decided it, and that the record has not been touched. Built for teams shipping AI in **finance, healthcare, legal, and any regulated industry** that needs to answer to auditors, regulators, or customers.

> EU AI Act · SOC 2 · HIPAA · GDPR · ISO 42001 — SealVera logs are designed to satisfy the explainability and auditability requirements of major AI compliance frameworks.

---

## Why SealVera?

- **Tamper-evident logs** — every decision is cryptographically hashed and chained; any tampering is detectable
- **Zero dependencies** — stdlib only, nothing added to your go.mod
- **Wrap any agent** — works with any LLM client: openai-go, anthropic-go, custom models
- **Full explainability** — captures inputs, outputs, reasoning, confidence scores, and model used
- **Real-time dashboard** — search, filter, and export your full AI decision history at [app.sealvera.com](https://app.sealvera.com)
- **Drift detection** — get alerted when agent behaviour deviates from its baseline
- **EU AI Act, HIPAA, GDPR, SOC 2** — built for regulated industries

---

## Installation

```bash
go get github.com/sealvera/sealvera-go
```

---

## Quick Start

```go
package main

import (
    "context"
    "github.com/sealvera/sealvera-go"
)

func main() {
    // Initialize once at startup
    sealvera.Init(sealvera.Config{
        Endpoint: "https://app.sealvera.com",
        APIKey:   "sv_your_api_key_here",
        Agent:    "payment-agent",
    })

    ctx := context.Background()

    // Wrap any agent function — input, output, and decision are logged automatically
    result, err := sealvera.Wrap(ctx, sealvera.WrapOptions{
        Agent:  "payment-agent",
        Action: "approve_payment",
        Input:  map[string]any{"amount": 5000, "currency": "USD", "customer_id": "c_123"},
    }, func() (any, error) {
        return processPayment(ctx, payment)
    })

    // result is logged with decision, reasoning, and cryptographic signature
    _ = result
    _ = err
}
```

Get your API key at **[app.sealvera.com](https://app.sealvera.com)**.

---

## API Reference

### `sealvera.Init(config)`

Initialize the SDK. Call once at application startup.

```go
sealvera.Init(sealvera.Config{
    Endpoint: "https://app.sealvera.com", // SealVera server URL (required)
    APIKey:   "sv_...",                    // API key from your dashboard (required)
    Agent:    "my-agent",                 // Default agent name for all logs
    Debug:    false,                       // Enable verbose debug logging
})
```

---

### `sealvera.Wrap(ctx, opts, fn)`

Wrap any agent function. Captures input, output, inferred decision, and timing.

```go
result, err := sealvera.Wrap(ctx, sealvera.WrapOptions{
    Agent:  "fraud-detector",
    Action: "evaluate_transaction",
    Input:  transaction,
}, func() (any, error) {
    // Your agent logic — LLM call, rules engine, ML model, anything
    return runFraudModel(ctx, transaction)
})
```

If the returned value contains a `Decision` field (`APPROVED`, `REJECTED`, `FLAGGED`), it is used as the decision label in the audit log.

---

### `sealvera.Log(ctx, entry)`

Manually log a decision entry.

```go
err := sealvera.Log(ctx, sealvera.LogEntry{
    Agent:    "underwriting-agent",
    Action:   "evaluate_loan",
    Decision: "APPROVED",
    Input:    application,
    Output:   result,
    Reasoning: []sealvera.ReasoningStep{
        {Factor: "credit_score", Value: "780", Signal: "safe", Explanation: "Above 700 threshold"},
        {Factor: "dti_ratio",   Value: "0.28", Signal: "safe", Explanation: "Below 0.43 limit"},
    },
    Confidence: 0.94,
})
```

---

### `sealvera.NewAgent(name)`

Create a named agent client for scoped logging.

```go
agent := sealvera.NewAgent("claims-processor")

result, err := agent.Wrap(ctx, sealvera.WrapOptions{
    Action: "triage_claim",
    Input:  claim,
}, func() (any, error) {
    return triageClaim(ctx, claim)
})
```

---

## Structured Decisions

For the richest audit trail, return a struct with a `Decision` field:

```go
type PaymentDecision struct {
    Decision   string  `json:"decision"`    // "APPROVED" | "REJECTED" | "FLAGGED"
    Reason     string  `json:"reason"`
    RiskScore  float64 `json:"risk_score"`
    Confidence float64 `json:"confidence"`
}

result, err := sealvera.Wrap(ctx, sealvera.WrapOptions{
    Agent:  "payment-agent",
    Action: "approve_payment",
    Input:  payment,
}, func() (any, error) {
    decision, err := runPaymentModel(ctx, payment)
    return PaymentDecision{
        Decision:   decision.Label,   // "APPROVED"
        Reason:     decision.Reason,
        RiskScore:  decision.Score,
        Confidence: decision.Confidence,
    }, err
})
// decision logged: APPROVED, full reasoning stored, cryptographically signed
```

---

## Decision Vocabulary

| Decision | Meaning | Use for |
|---|---|---|
| `APPROVED` | Request approved | Payments, loans, access grants |
| `REJECTED` | Request blocked | Fraud blocks, denials |
| `FLAGGED` | Needs human review | Borderline cases |
| `COMPLETED` | Task finished | General agent tasks |
| `FAILED` | Task failed | Error paths |
| `ESCALATED` | Handed to human | Human-in-the-loop |

---

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `SEALVERA_ENDPOINT` | SealVera server URL | `https://app.sealvera.com` |
| `SEALVERA_API_KEY` | Your API key (starts with `sv_`) | — |
| `SEALVERA_AGENT` | Default agent name | `default` |
| `SEALVERA_DEBUG` | Enable debug logging | `false` |

```go
// Config from environment
sealvera.InitFromEnv()
```

---

## Use Cases

- **Financial services** — log every credit decision, fraud flag, and payment approval for FINRA/OCC review
- **Healthcare AI** — tamper-evident audit trail for clinical decision support (HIPAA-aligned)
- **Legal tech** — record document review, contract analysis, and compliance risk assessments
- **Insurance** — log claims triage, underwriting decisions, and anomaly flags
- **Any agentic AI system** — multi-step reasoning chains, tool calls, and autonomous decisions

---

## Links

- **Dashboard & signup** — [app.sealvera.com](https://app.sealvera.com)
- **Full documentation** — [app.sealvera.com/docs](https://app.sealvera.com/docs)
- **Node.js SDK** — [github.com/SealVera/sealvera-js](https://github.com/SealVera/sealvera-js)
- **Python SDK** — [github.com/SealVera/sealvera-python](https://github.com/SealVera/sealvera-python)
- **Support** — [hello@sealvera.com](mailto:hello@sealvera.com)

---

## License

MIT — see [LICENSE](./LICENSE)
