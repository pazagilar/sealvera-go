package sealvera

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the SealVera HTTP client.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a new SealVera client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendLog sends a single log entry to the SealVera server.
func (c *Client) SendLog(ctx context.Context, entry LogEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("sealvera: failed to marshal log entry: %w", err)
	}

	endpoint := strings.TrimRight(c.cfg.Endpoint, "/")
	url := endpoint + "/api/ingest"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("sealvera: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SealVera-Key", c.cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sealvera: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sealvera: server returned %d: %s", resp.StatusCode, string(body))
	}

	if c.cfg.Debug {
		fmt.Printf("[SealVera] Sent log: %s → %s\n", entry.Action, entry.Decision)
	}

	return nil
}

// sendLogAsync sends a log entry asynchronously (non-blocking, non-fatal).
func (c *Client) sendLogAsync(entry LogEntry) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.SendLog(ctx, entry); err != nil {
			if c.cfg.Debug {
				fmt.Printf("[SealVera] Log send failed (non-fatal): %v\n", err)
			}
		}
	}()
}
