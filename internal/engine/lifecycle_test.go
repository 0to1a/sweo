package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebhookPayloadSerialization(t *testing.T) {
	payload := WebhookPayload{
		Event:      "status_change",
		SessionID:  "ao-1",
		IssueID:    "42",
		IssueTitle: "Fix login bug",
		OldStatus:  "working",
		NewStatus:  "pr_open",
		PRURL:      "https://github.com/org/repo/pull/99",
		Timestamp:  "2026-04-04T10:30:00Z",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)

	var decoded WebhookPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestFireWebhook(t *testing.T) {
	received := make(chan WebhookPayload, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "myproject", r.URL.Query().Get("project"))

		var payload WebhookPayload
		json.NewDecoder(r.Body).Decode(&payload)
		received <- payload
		w.WriteHeader(200)
	}))
	defer server.Close()

	payload := WebhookPayload{
		Event:     "status_change",
		SessionID: "ao-1",
		OldStatus: "working",
		NewStatus: "pr_open",
		Timestamp: "2026-04-04T10:30:00Z",
	}

	FireWebhook(server.URL, "myproject", payload)

	select {
	case got := <-received:
		assert.Equal(t, "ao-1", got.SessionID)
		assert.Equal(t, "pr_open", got.NewStatus)
	case <-time.After(3 * time.Second):
		t.Fatal("webhook not received within timeout")
	}
}

func TestFireWebhookEmptyURL(t *testing.T) {
	// Should not panic with empty URL
	FireWebhook("", "project", WebhookPayload{})
}
