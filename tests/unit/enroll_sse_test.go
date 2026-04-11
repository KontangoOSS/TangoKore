package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KontangoOSS/TangoKore/internal/enroll"
)

func TestSSEEnroll_NewMachine(t *testing.T) {
	// Mock SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// Send verify events
		checks := []struct {
			check      string
			passed     bool
			confidence string
		}{
			{"fingerprint_match", false, "unknown"},
			{"os_validation", true, "high"},
			{"banned_check", true, "high"},
		}

		for _, c := range checks {
			data := map[string]interface{}{
				"check":      c.check,
				"passed":     c.passed,
				"confidence": c.confidence,
			}
			dataBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
			flusher.Flush()
		}

		// Send decision event
		decision := map[string]interface{}{
			"status": "quarantine",
			"reason": "new machine",
		}
		decisionBytes, _ := json.Marshal(decision)
		fmt.Fprintf(w, "event: decision\ndata: %s\n\n", string(decisionBytes))
		flusher.Flush()

		// Send identity event
		identity := map[string]interface{}{
			"id":       "machine-123",
			"nickname": "test-machine",
			"status":   "quarantine",
			"identity": json.RawMessage(`{"type":"pkcs12"}`),
			"config": map[string]interface{}{
				"hosts":  []string{"ziti.example.com"},
				"tunnel": map[string]interface{}{},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()
	}))
	defer server.Close()

	result, err := enroll.SSEEnroll(server.URL, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "machine-123" {
		t.Errorf("expected ID 'machine-123', got %q", result.ID)
	}
	if result.Nickname != "test-machine" {
		t.Errorf("expected nickname 'test-machine', got %q", result.Nickname)
	}
	if result.Status != "quarantine" {
		t.Errorf("expected status 'quarantine', got %q", result.Status)
	}
}

func TestSSEEnroll_RejectedMachine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// Send decision event with rejected status
		decision := map[string]interface{}{
			"status": "rejected",
			"reason": "banned hardware",
		}
		decisionBytes, _ := json.Marshal(decision)
		fmt.Fprintf(w, "event: decision\ndata: %s\n\n", string(decisionBytes))
		flusher.Flush()
	}))
	defer server.Close()

	result, err := enroll.SSEEnroll(server.URL, "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("expected error containing 'rejected', got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestSSEEnroll_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// Send error event
		errEvent := map[string]interface{}{
			"reason": "internal server error",
		}
		errBytes, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(errBytes))
		flusher.Flush()
	}))
	defer server.Close()

	result, err := enroll.SSEEnroll(server.URL, "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "internal server error") {
		t.Errorf("expected error about internal server error, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestSSEEnroll_NoIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send no events, just close
	}))
	defer server.Close()

	result, err := enroll.SSEEnroll(server.URL, "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no identity") {
		t.Errorf("expected error about no identity, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestSSEEnroll_HTTP404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result, err := enroll.SSEEnroll(server.URL, "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected HTTP 404 error, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestSSEEnrollStream_EventCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// Send one verify event
		data := map[string]interface{}{
			"check":      "fingerprint",
			"passed":     true,
			"confidence": "high",
		}
		dataBytes, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
		flusher.Flush()

		// Send identity event
		identity := map[string]interface{}{
			"id":       "machine-456",
			"nickname": "callback-test",
			"status":   "approved",
			"identity": json.RawMessage(`{"type":"pkcs12"}`),
			"config": map[string]interface{}{
				"hosts":  []string{},
				"tunnel": map[string]interface{}{},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()
	}))
	defer server.Close()

	var eventsSeen []enroll.SSEEvent
	eventFn := func(evt enroll.SSEEvent) {
		eventsSeen = append(eventsSeen, evt)
	}

	result, err := enroll.SSEEnrollStream(server.URL, "new", "", "", "", "", eventFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "machine-456" {
		t.Errorf("expected ID 'machine-456', got %q", result.ID)
	}

	// Check that verify event was seen
	verifyEventSeen := false
	for _, evt := range eventsSeen {
		if evt.Kind == "verify" {
			verifyEventSeen = true
			if evt.Check != "fingerprint" {
				t.Errorf("expected check 'fingerprint', got %q", evt.Check)
			}
			if !evt.Passed {
				t.Errorf("expected passed=true, got false")
			}
		}
	}
	if !verifyEventSeen {
		t.Errorf("expected verify event in callback, got events: %v", eventsSeen)
	}
}

func TestSSEEnrollStream_WithProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify profile was sent in the request
		decoder := json.NewDecoder(r.Body)
		var payload map[string]interface{}
		decoder.Decode(&payload)

		if profile, ok := payload["profile"].(string); !ok || profile != "stage-1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// Send identity event
		identity := map[string]interface{}{
			"id":       "profile-test",
			"nickname": "profile-machine",
			"status":   "approved",
			"identity": json.RawMessage(`{}`),
			"config": map[string]interface{}{
				"hosts":  []string{},
				"tunnel": map[string]interface{}{},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()
	}))
	defer server.Close()

	result, err := enroll.SSEEnrollStream(server.URL, "new", "", "", "", "stage-1", func(enroll.SSEEvent) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "profile-test" {
		t.Errorf("expected ID 'profile-test', got %q", result.ID)
	}
}
