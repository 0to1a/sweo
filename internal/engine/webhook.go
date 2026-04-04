package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const webhookTimeout = 5 * time.Second

// WebhookPayload is sent on every session status change.
type WebhookPayload struct {
	Event     string `json:"event"`
	SessionID string `json:"session_id"`
	IssueID   string `json:"issue_id"`
	IssueTitle string `json:"issue_title"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	PRURL     string `json:"pr_url,omitempty"`
	Timestamp string `json:"timestamp"`
}

// FireWebhook sends a status change notification to the configured webhook URL.
// Fire-and-forget: errors are logged but don't block.
func FireWebhook(webhookURL, projectName string, payload WebhookPayload) {
	if webhookURL == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
		defer cancel()

		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("WARN: webhook marshal error: %v", err)
			return
		}

		url := fmt.Sprintf("%s?project=%s", webhookURL, projectName)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			log.Printf("WARN: webhook request error: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("WARN: webhook delivery failed for %s: %v", payload.SessionID, err)
			return
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			log.Printf("WARN: webhook returned %d for %s", resp.StatusCode, payload.SessionID)
		}
	}()
}
