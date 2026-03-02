// SealVera Go SDK — Payment Agent Example
//
// Demonstrates how to use SealVera to audit AI-powered payment decisions.
//
// Prerequisites:
//   - A running SealVera server (npm start in the sealvera directory)
//   - SEALVERA_API_KEY env var set to the key printed on startup
//
// Run:
//
//	cd examples
//	SEALVERA_API_KEY=sv_... go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	sealvera "github.com/sealvera/sealvera-go"
)

// Payment represents a financial transaction to be analyzed
type Payment struct {
	TransactionID   string          `json:"transaction_id"`
	Amount          float64         `json:"amount"`
	Currency        string          `json:"currency"`
	CustomerID      string          `json:"customer_id"`
	Merchant        string          `json:"merchant"`
	Category        string          `json:"category"`
	CustomerHistory CustomerHistory `json:"customer_history"`
}

type CustomerHistory struct {
	AvgTransaction float64 `json:"avg_transaction"`
	AccountAgeDays int     `json:"account_age_days"`
	FraudScore     float64 `json:"fraud_score"`
}

// PaymentDecision is the AI agent's response
type PaymentDecision struct {
	Decision    string   `json:"decision"`
	Reason      string   `json:"reason"`
	RiskFactors []string `json:"risk_factors"`
	Confidence  float64  `json:"confidence"`
}

func main() {
	// Initialize SealVera
	endpoint := getEnv("SEALVERA_ENDPOINT", "http://localhost:3000")
	apiKey := os.Getenv("SEALVERA_API_KEY")

	if apiKey == "" {
		fmt.Println("⚠️  SEALVERA_API_KEY not set. Set it to the key printed by the SealVera server on startup.")
		fmt.Println("   Example: SEALVERA_API_KEY=sv_... go run main.go")
		os.Exit(1)
	}

	err := sealvera.Init(sealvera.Config{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Agent:    "payment-agent",
		Debug:    true,
	})
	if err != nil {
		fmt.Printf("Failed to initialize SealVera: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🔍 SealVera Payment Agent Demo (Go)\n")
	fmt.Printf("Dashboard: %s\n\n", endpoint)

	// Sample payments to process
	payments := []Payment{
		{
			TransactionID: "txn_001",
			Amount:        150.00,
			Currency:      "USD",
			CustomerID:    "cust_abc123",
			Merchant:      "Amazon",
			Category:      "retail",
			CustomerHistory: CustomerHistory{
				AvgTransaction: 85.0,
				AccountAgeDays: 730,
				FraudScore:     0.02,
			},
		},
		{
			TransactionID: "txn_002",
			Amount:        8500.00,
			Currency:      "USD",
			CustomerID:    "cust_xyz789",
			Merchant:      "Unknown International Merchant",
			Category:      "international",
			CustomerHistory: CustomerHistory{
				AvgTransaction: 200.0,
				AccountAgeDays: 30,
				FraudScore:     0.78,
			},
		},
		{
			TransactionID: "txn_003",
			Amount:        42.50,
			Currency:      "USD",
			CustomerID:    "cust_def456",
			Merchant:      "Whole Foods",
			Category:      "grocery",
			CustomerHistory: CustomerHistory{
				AvgTransaction: 65.0,
				AccountAgeDays: 1200,
				FraudScore:     0.01,
			},
		},
	}

	ctx := context.Background()

	for _, payment := range payments {
		fmt.Printf("Processing transaction: %s\n", payment.TransactionID)
		fmt.Printf("  Amount: $%.2f at %s\n", payment.Amount, payment.Merchant)

		result, err := sealvera.Wrap(ctx, sealvera.WrapOptions{
			Agent:  "payment-agent",
			Action: "approve_payment",
			Input:  payment,
		}, func() (any, error) {
			return simulatePaymentDecision(payment)
		})

		if err != nil {
			fmt.Printf("  Error: %v\n\n", err)
			continue
		}

		// Parse result
		var decision PaymentDecision
		b, _ := json.Marshal(result)
		json.Unmarshal(b, &decision)

		icon := "✅"
		if decision.Decision == "REJECTED" {
			icon = "❌"
		} else if decision.Decision == "FLAGGED" {
			icon = "⚠️"
		}

		fmt.Printf("  %s %s (confidence: %.0f%%)\n", icon, decision.Decision, decision.Confidence*100)
		fmt.Printf("  Reason: %s\n\n", decision.Reason)
	}

	// Give async logs time to send
	time.Sleep(1 * time.Second)

	fmt.Println("✓ Done! Check the dashboard to see all logged decisions.")
	fmt.Printf("  → %s\n\n", endpoint)
}

// simulatePaymentDecision mimics an AI agent making a payment decision.
// In a real implementation, this would call OpenAI, Anthropic, etc.
func simulatePaymentDecision(payment Payment) (PaymentDecision, error) {
	// Rule-based simulation (replace with real LLM call)
	decision := PaymentDecision{
		Confidence: 0.92,
	}

	// High fraud score
	if payment.CustomerHistory.FraudScore > 0.5 {
		decision.Decision = "REJECTED"
		decision.Reason = fmt.Sprintf("High fraud score (%.2f) exceeds threshold. Amount $%.2f significantly above customer average ($%.2f).",
			payment.CustomerHistory.FraudScore, payment.Amount, payment.CustomerHistory.AvgTransaction)
		decision.RiskFactors = []string{"high_fraud_score", "unusual_amount", "new_account"}
		decision.Confidence = 0.95
		return decision, nil
	}

	// Amount is reasonable relative to history
	if payment.Amount <= payment.CustomerHistory.AvgTransaction*3 && payment.CustomerHistory.FraudScore < 0.1 {
		decision.Decision = "APPROVED"
		decision.Reason = fmt.Sprintf("Transaction amount $%.2f within normal range. Low fraud score (%.2f). Established account.",
			payment.Amount, payment.CustomerHistory.FraudScore)
		decision.RiskFactors = []string{}
		return decision, nil
	}

	// Borderline case
	decision.Decision = "FLAGGED"
	decision.Reason = fmt.Sprintf("Transaction requires manual review. Amount $%.2f is %.1fx above customer average.",
		payment.Amount, payment.Amount/payment.CustomerHistory.AvgTransaction)
	decision.RiskFactors = []string{"unusual_amount"}
	decision.Confidence = 0.75
	return decision, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
