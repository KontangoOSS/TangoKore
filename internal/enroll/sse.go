package enroll

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// EnrollResult is the outcome of an SSE enrollment.
type EnrollResult struct {
	Status   string          `json:"status"`
	ID       string          `json:"id"`
	Nickname string          `json:"nickname"`
	Identity json.RawMessage `json:"identity"`
	Config   struct {
		Hosts  []string               `json:"hosts"`
		Tunnel map[string]interface{} `json:"tunnel"`
	} `json:"config"`
	Reason string `json:"reason,omitempty"`
}

// SSEEvent is a single event from the enrollment stream.
type SSEEvent struct {
	Kind       string // verify, progress, decision, identity, error
	Check      string // for verify events
	Passed     bool   // for verify events
	Confidence string // for verify events
	Step       string // for progress events
	Status     string // for decision / identity events
	Reason     string // for decision / error events
	Result     *EnrollResult // populated on identity event
}

// SSEEnrollStream is like SSEEnroll but emits events to eventFn as they arrive.
// Returns the final EnrollResult when the identity event is received.
// eventFn may be called from a goroutine; it must be safe to call concurrently.
//
// The endpoint always receives the same message format (machine data).
// The server determines the method (new/scan/trusted) based on:
// - Fingerprint matching (does it have history?)
// - Credentials provided (AppRole, JWT, etc.)
// - Server policy
//
// Deprecated parameters (method, session, roleID, secretID, profile) are kept
// for backward compatibility but should not be used. All enrollment goes through
// the same endpoint with the same message format.
func SSEEnrollStream(url, method, session, roleID, secretID, profile string, eventFn func(SSEEvent)) (*EnrollResult, error) {
	os := ProbeOS()
	hw := ProbeHardware()
	net := ProbeNetwork()
	sys := ProbeSystem()

	// All machines send the same message format to the same endpoint.
	// The server determines what method applies (new/scan/trusted/etc)
	// based on the data and its own policy.
	payload := map[string]interface{}{}

	// Always send machine data
	for _, m := range []map[string]interface{}{os, hw, net, sys} {
		for k, v := range m {
			if k != "type" {
				payload[k] = v
			}
		}
	}

	// Optionally send credentials if provided (server will validate)
	if roleID != "" && secretID != "" {
		payload["role_id"] = roleID
		payload["secret_id"] = secretID
	}

	// Optionally send session token if provided
	if session != "" {
		payload["session"] = session
	}

	// Optionally send profile preference if provided
	if profile != "" {
		payload["profile"] = profile
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url+"/api/enroll/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var result *EnrollResult
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		switch currentEvent {
		case "verify":
			var v struct {
				Check      string `json:"check"`
				Passed     bool   `json:"passed"`
				Confidence string `json:"confidence"`
				Reason     string `json:"reason"`
			}
			json.Unmarshal([]byte(data), &v)
			if eventFn != nil {
				eventFn(SSEEvent{Kind: "verify", Check: v.Check, Passed: v.Passed, Confidence: v.Confidence, Reason: v.Reason})
			}
		case "progress":
			var p struct{ Step string `json:"step"` }
			json.Unmarshal([]byte(data), &p)
			if eventFn != nil {
				eventFn(SSEEvent{Kind: "progress", Step: p.Step})
			}
		case "decision":
			var d struct {
				Status string `json:"status"`
				Reason string `json:"reason"`
			}
			json.Unmarshal([]byte(data), &d)
			if eventFn != nil {
				eventFn(SSEEvent{Kind: "decision", Status: d.Status, Reason: d.Reason})
			}
			if d.Status == "rejected" {
				return nil, fmt.Errorf("rejected: %s", d.Reason)
			}
		case "identity":
			result = &EnrollResult{}
			var id struct {
				ID       string          `json:"id"`
				Nickname string          `json:"nickname"`
				Status   string          `json:"status"`
				Identity json.RawMessage `json:"identity"`
				Config   struct {
					Hosts  []string               `json:"hosts"`
					Tunnel map[string]interface{} `json:"tunnel"`
				} `json:"config"`
			}
			json.Unmarshal([]byte(data), &id)
			result.ID = id.ID
			result.Nickname = id.Nickname
			result.Status = id.Status
			result.Identity = id.Identity
			result.Config.Hosts = id.Config.Hosts
			result.Config.Tunnel = id.Config.Tunnel
			if eventFn != nil {
				eventFn(SSEEvent{Kind: "identity", Status: id.Status, Result: result})
			}
		case "error":
			var e struct{ Reason string `json:"reason"` }
			json.Unmarshal([]byte(data), &e)
			if eventFn != nil {
				eventFn(SSEEvent{Kind: "error", Reason: e.Reason})
			}
			return nil, fmt.Errorf("server: %s", e.Reason)
		}
		currentEvent = ""
	}

	if result == nil {
		return nil, fmt.Errorf("no identity received")
	}
	return result, nil
}

// SSEEnroll sends all probe data in one POST and reads SSE events back.
// This is the enrollment path — one request, streaming response.
// It calls SSEEnrollStream with a logging callback for verbose output.
//
// The enrollment endpoint receives machine data and the server determines
// whether this is a new, returning, or trusted machine based on:
// - Machine fingerprint (has it enrolled before?)
// - Credentials (AppRole, JWT, session token)
// - Server policy
//
// SSEEnroll is a convenience wrapper that calls SSEEnrollStream with logging.
// The enrollment method is determined by the server based on credentials.
func SSEEnroll(url, session, roleID, secretID string) (*EnrollResult, error) {
	// Collect probe data to log before sending
	payload := map[string]interface{}{
		"session": session,
	}
	if roleID != "" {
		payload["role_id"] = roleID
		payload["secret_id"] = secretID
	}
	// Merge all probe data to get it for logging
	for _, m := range []map[string]interface{}{ProbeOS(), ProbeHardware(), ProbeNetwork(), ProbeSystem()} {
		for k, v := range m {
			if k != "type" {
				payload[k] = v
			}
		}
	}

	// Log machine info
	log.Printf("  hostname:    %s", payload["hostname"])
	log.Printf("  os:          %s", payload["os_version"])
	log.Printf("  arch:        %s", payload["arch"])
	if hw, ok := payload["hardware_hash"].(string); ok && hw != "" {
		log.Printf("  fingerprint: %s", hw)
	}

	// Call SSEEnrollStream with a logging callback
	return SSEEnrollStream(url, "", session, roleID, secretID, "", func(evt SSEEvent) {
		switch evt.Kind {
		case "verify":
			status := "✓"
			if !evt.Passed {
				status = "✗"
			}
			log.Printf("  verify: %s %s", evt.Check, status)
		case "progress":
			log.Printf("  %s…", evt.Step)
		case "decision":
			log.Printf("  decision: %s (%s)", evt.Status, evt.Reason)
		}
	})
}
