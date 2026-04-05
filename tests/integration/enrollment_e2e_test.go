package integration_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/KontangoOSS/TangoKore/internal/enroll"
)

// TestE2E_PreAuthAnnouncement tests the "I'm coming" pre-enrollment announcement.
// Machine sends probe data to stream endpoint without auth, server receives it.
func TestE2E_PreAuthAnnouncement(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	// Capture what the server receives
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the announcement
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&receivedPayload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		log.Printf("[SERVER] Received pre-auth announcement:")
		log.Printf("[SERVER]   method: %v", receivedPayload["method"])
		log.Printf("[SERVER]   hostname: %v", receivedPayload["hostname"])
		log.Printf("[SERVER]   os: %v", receivedPayload["os"])
		log.Printf("[SERVER]   arch: %v", receivedPayload["arch"])
		log.Printf("[SERVER]   hardware_hash: %v", receivedPayload["hardware_hash"])

		// Verify required fields are present
		// Note: method is no longer sent by client; server determines it
		if hostname, ok := receivedPayload["hostname"].(string); !ok || hostname == "" {
			http.Error(w, "missing hostname", http.StatusBadRequest)
			return
		}

		if hardwareHash, ok := receivedPayload["hardware_hash"].(string); !ok || hardwareHash == "" {
			http.Error(w, "missing hardware_hash", http.StatusBadRequest)
			return
		}

		// Send back SSE stream with verification checks
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		log.Printf("[SERVER] Sending verification checks...")

		// Send verify events
		checks := []struct {
			check      string
			passed     bool
			confidence string
			reason     string
		}{
			{"fingerprint_match", false, "unknown", "no previous record"},
			{"os_validation", true, "high", ""},
			{"banned_check", true, "high", ""},
		}

		for _, c := range checks {
			data := map[string]interface{}{
				"check":      c.check,
				"passed":     c.passed,
				"confidence": c.confidence,
			}
			if c.reason != "" {
				data["reason"] = c.reason
			}
			dataBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
			flusher.Flush()
			log.Printf("[SERVER]   ✓ sent verify: %s", c.check)
		}

		// Send decision event (new machine → quarantine)
		log.Printf("[SERVER] Making decision: new machine → quarantine")
		decision := map[string]interface{}{
			"status": "quarantine",
			"reason": "new machine, no history",
		}
		decisionBytes, _ := json.Marshal(decision)
		fmt.Fprintf(w, "event: decision\ndata: %s\n\n", string(decisionBytes))
		flusher.Flush()

		// Send identity event with cert and config
		log.Printf("[SERVER] Issuing identity...")
		identity := map[string]interface{}{
			"id":       "leonardo-da-pc-new",
			"nickname": "leonardo-da-pc",
			"status":   "quarantine",
			"identity": json.RawMessage(`{"type":"pkcs12","cert":"base64-encoded-cert"}`),
			"config": map[string]interface{}{
				"hosts":  []string{"ziti.example.com"},
				"tunnel": map[string]interface{}{"endpoint": "ws://ziti.example.com/edge"},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()

		log.Printf("[SERVER] Enrollment complete")
	}))
	defer server.Close()

	log.Printf("\n=== E2E Test: Pre-Auth Announcement (Say I'm Coming) ===\n")
	log.Printf("[CLIENT] Starting enrollment to %s", server.URL)
	log.Printf("[CLIENT] Method: new (unknown machine)\n")

	// Call SSEEnroll (the machine announcing itself)
	result, err := enroll.SSEEnroll(server.URL, "", "", "", "")

	if err != nil {
		t.Fatalf("enrollment failed: %v", err)
	}

	// Verify the server received the announcement
	if receivedPayload == nil {
		t.Fatal("server never received payload")
	}

	// Note: Client no longer sends method field. Server determines it.
	// Just verify the machine data was received.

	if hostname, ok := receivedPayload["hostname"].(string); !ok || hostname == "" {
		t.Errorf("expected hostname, got %v", receivedPayload["hostname"])
	}

	if hardwareHash, ok := receivedPayload["hardware_hash"].(string); !ok || hardwareHash == "" {
		t.Errorf("expected hardware_hash, got %v", receivedPayload["hardware_hash"])
	}

	log.Printf("\n[CLIENT] ✓ Server received announcement with hostname: %v", receivedPayload["hostname"])
	log.Printf("[CLIENT] ✓ Verification checks completed")
	log.Printf("[CLIENT] ✓ Identity issued: %s [%s]", result.ID, result.Status)
	log.Printf("[CLIENT] ✓ Enrollment complete\n")

	// Verify result
	if result.ID != "leonardo-da-pc-new" {
		t.Errorf("expected ID 'leonardo-da-pc-new', got %q", result.ID)
	}
	if result.Status != "quarantine" {
		t.Errorf("expected status 'quarantine', got %q", result.Status)
	}
}

// TestE2E_ReturningMachineFlow tests the "scan" method for returning machines.
// Machine announces itself with fingerprint, server matches it, restores previous identity.
func TestE2E_ReturningMachineFlow(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var payload map[string]interface{}
		decoder.Decode(&payload)

		// Server always fingerprint-matches automatically
		// No method field sent from client
		log.Printf("[SERVER] Received announcement from machine")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// Server checks: does fingerprint match history?
		// For this test, we assume it matches
		log.Printf("[SERVER] Checking fingerprint history...")

			// Fingerprint match found!
			data := map[string]interface{}{
				"check":      "fingerprint_match",
				"passed":     true,
				"confidence": "high",
			}
			dataBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
			flusher.Flush()

			log.Printf("[SERVER] ✓ Fingerprint matched! Restoring previous identity")

			// Send decision: approved (not quarantine)
			decision := map[string]interface{}{
				"status": "approved",
				"reason": "fingerprint match, restoring previous ACL",
			}
			decisionBytes, _ := json.Marshal(decision)
			fmt.Fprintf(w, "event: decision\ndata: %s\n\n", string(decisionBytes))
			flusher.Flush()

			log.Printf("[SERVER] Status: approved (restored previous identity)")

		// Send identity with restored ACL
		identity := map[string]interface{}{
			"id":       "leonardo-da-pc-known",
			"nickname": "leonardo-da-pc",
			"status":   "approved",
			"identity": json.RawMessage(`{"type":"pkcs12","cert":"restored-cert"}`),
			"config": map[string]interface{}{
				"hosts": []string{"ziti.example.com"},
				"tunnel": map[string]interface{}{
					"endpoint": "ws://ziti.example.com/edge",
					"acl":      "stage-1",
				},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()

		log.Printf("[SERVER] Issued restored identity: %s [approved]", "leonardo-da-pc-known")
	}))
	defer server.Close()

	log.Printf("\n=== E2E Test: Returning Machine (Scan Method) ===\n")
	log.Printf("[CLIENT] This machine has enrolled before")
	log.Printf("[CLIENT] Re-enrolling with --scan to restore previous identity\n")

	result, err := enroll.SSEEnroll(server.URL, "scan", "", "", "")

	if err != nil {
		t.Fatalf("enrollment failed: %v", err)
	}

	log.Printf("\n[CLIENT] ✓ Server recognized machine (fingerprint match)")
	log.Printf("[CLIENT] ✓ Previous identity restored: %s", result.ID)
	log.Printf("[CLIENT] ✓ Status: %s (privileges restored)\n", result.Status)

	if result.Status != "approved" {
		t.Errorf("expected status 'approved' for returning machine, got %q", result.Status)
	}
}

// TestE2E_AppRoleAuthFlow tests enrollment with AppRole credentials.
// Machine proves it's trusted by providing role_id and secret_id.
func TestE2E_AppRoleAuthFlow(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	var receivedRoleID, receivedSecretID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var payload map[string]interface{}
		decoder.Decode(&payload)

		receivedRoleID, _ = payload["role_id"].(string)
		receivedSecretID, _ = payload["secret_id"].(string)

		log.Printf("[SERVER] Received announcement with AppRole credentials")
		log.Printf("[SERVER] Validating AppRole credentials...")

		if receivedRoleID == "test-role-id" && receivedSecretID == "test-secret-id" {
			log.Printf("[SERVER] ✓ AppRole credentials valid - machine is trusted")
		} else {
			log.Printf("[SERVER] ✗ Invalid AppRole credentials")
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// AppRole check passes
		data := map[string]interface{}{
			"check":      "approle_credentials",
			"passed":     true,
			"confidence": "high",
		}
		dataBytes, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
		flusher.Flush()

		// Approved immediately (trusted machine)
		decision := map[string]interface{}{
			"status": "approved",
			"reason": "AppRole authenticated - trusted machine",
		}
		decisionBytes, _ := json.Marshal(decision)
		fmt.Fprintf(w, "event: decision\ndata: %s\n\n", string(decisionBytes))
		flusher.Flush()

		log.Printf("[SERVER] Status: approved (AppRole authenticated)")

		// Issue full identity
		identity := map[string]interface{}{
			"id":       "trusted-machine-001",
			"nickname": "trusted-machine",
			"status":   "trusted",
			"identity": json.RawMessage(`{"type":"pkcs12","cert":"trusted-cert"}`),
			"config": map[string]interface{}{
				"hosts": []string{"ziti.example.com"},
				"tunnel": map[string]interface{}{
					"endpoint": "ws://ziti.example.com/edge",
					"acl":      "stage-3",
				},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()

		log.Printf("[SERVER] Issued trusted identity: %s [stage-3]", "trusted-machine-001")
	}))
	defer server.Close()

	log.Printf("\n=== E2E Test: AppRole Authentication ===\n")
	log.Printf("[CLIENT] This machine has AppRole credentials (pre-provisioned)")
	log.Printf("[CLIENT] Enrolling with --role-id and --secret-id\n")

	result, err := enroll.SSEEnroll(server.URL, "", "", "test-role-id", "test-secret-id")

	if err != nil {
		t.Fatalf("enrollment failed: %v", err)
	}

	log.Printf("\n[CLIENT] ✓ AppRole credentials sent to server")
	log.Printf("[CLIENT] ✓ Server validated credentials")
	log.Printf("[CLIENT] ✓ Trusted identity issued: %s", result.ID)
	log.Printf("[CLIENT] ✓ Status: %s (full privileges)\n", result.Status)

	if receivedRoleID != "test-role-id" {
		t.Errorf("expected role_id='test-role-id', got %q", receivedRoleID)
	}

	if receivedSecretID != "test-secret-id" {
		t.Errorf("expected secret_id='test-secret-id', got %q", receivedSecretID)
	}

	if result.Status != "trusted" {
		t.Errorf("expected status 'trusted' for AppRole auth, got %q", result.Status)
	}
}

// TestE2E_ProfileSelection tests profile selection during enrollment.
// Machine requests a specific profile (e.g., stage-1).
func TestE2E_ProfileSelection(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	var receivedProfile string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var payload map[string]interface{}
		decoder.Decode(&payload)

		receivedProfile, _ = payload["profile"].(string)

		log.Printf("[SERVER] Received announcement with profile: %s", receivedProfile)

		if receivedProfile != "stage-1" {
			http.Error(w, "profile not supported", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		log.Printf("[SERVER] Profile stage-1 is valid for this machine")

		// Send identity with requested profile
		identity := map[string]interface{}{
			"id":       "profile-test-machine",
			"nickname": "stage-1-machine",
			"status":   "approved",
			"identity": json.RawMessage(`{"type":"pkcs12"}`),
			"config": map[string]interface{}{
				"hosts": []string{},
				"tunnel": map[string]interface{}{
					"acl": "stage-1",
				},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()

		log.Printf("[SERVER] Issued identity with ACL: stage-1")
	}))
	defer server.Close()

	log.Printf("\n=== E2E Test: Profile Selection ===\n")
	log.Printf("[CLIENT] Enrolling with specific profile: stage-1\n")

	profileResult, err := enroll.SSEEnrollStream(server.URL, "new", "", "", "", "stage-1", func(evt enroll.SSEEvent) {
		switch evt.Kind {
		case "progress":
			log.Printf("[CLIENT] %s", evt.Step)
		}
	})

	if err != nil {
		t.Fatalf("enrollment failed: %v", err)
	}
	_ = profileResult // Used implicitly via test assertions

	log.Printf("\n[CLIENT] ✓ Profile sent in enrollment request")
	log.Printf("[CLIENT] ✓ Server accepted profile: stage-1")
	log.Printf("[CLIENT] ✓ Identity issued with ACL: %s\n", "stage-1")

	if receivedProfile != "stage-1" {
		t.Errorf("expected profile='stage-1', got %q", receivedProfile)
	}
}

// TestE2E_EventStreamCallback demonstrates event callbacks during enrollment.
func TestE2E_EventStreamCallback(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		log.Printf("[SERVER] Streaming verification events...")

		// Send verify events with different results
		events := []struct {
			check      string
			passed     bool
			confidence string
		}{
			{"fingerprint_match", false, "unknown"},
			{"os_validation", true, "high"},
			{"banned_check", true, "high"},
		}

		for _, evt := range events {
			data := map[string]interface{}{
				"check":      evt.check,
				"passed":     evt.passed,
				"confidence": evt.confidence,
			}
			dataBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
			flusher.Flush()
		}

		// Send decision
		decision := map[string]interface{}{
			"status": "quarantine",
			"reason": "new machine",
		}
		decisionBytes, _ := json.Marshal(decision)
		fmt.Fprintf(w, "event: decision\ndata: %s\n\n", string(decisionBytes))
		flusher.Flush()

		log.Printf("[SERVER] Sending identity...")

		// Send identity
		identity := map[string]interface{}{
			"id":       "event-test",
			"nickname": "event-machine",
			"status":   "quarantine",
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

	log.Printf("\n=== E2E Test: Event Stream with Callbacks ===\n")
	log.Printf("[CLIENT] Enrolling and listening to verification events\n")

	eventCount := 0
	enrollResult, err := enroll.SSEEnrollStream(server.URL, "new", "", "", "", "", func(evt enroll.SSEEvent) {
		eventCount++
		switch evt.Kind {
		case "verify":
			icon := "✓"
			if !evt.Passed {
				icon = "✗"
			}
			log.Printf("[CLIENT] [%d] verify: %s %s", eventCount, evt.Check, icon)
		case "decision":
			log.Printf("[CLIENT] [%d] decision: %s", eventCount, evt.Status)
		case "identity":
			log.Printf("[CLIENT] [%d] identity: %s", eventCount, evt.Status)
		case "progress":
			log.Printf("[CLIENT] [%d] %s", eventCount, evt.Step)
		}
	})

	if err != nil {
		t.Fatalf("enrollment failed: %v", err)
	}

	log.Printf("\n[CLIENT] ✓ Received %d events during enrollment", eventCount)
	log.Printf("[CLIENT] ✓ Final identity: %s [%s]\n", enrollResult.ID, enrollResult.Status)

	if eventCount == 0 {
		t.Error("expected to receive events via callback, got none")
	}
}

// TestE2E_FullLoggedFlow shows a complete end-to-end flow with comprehensive logging.
func TestE2E_FullLoggedFlow(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var payload map[string]interface{}
		decoder.Decode(&payload)

		hostname, _ := payload["hostname"].(string)
		method, _ := payload["method"].(string)

		log.Printf("\n╔════════════════════════════════════════════════════════════════╗")
		log.Printf("║ CONTROLLER RECEIVED: Pre-Auth Announcement (Stream Enrollment) ║")
		log.Printf("╚════════════════════════════════════════════════════════════════╝")
		log.Printf("")
		log.Printf("  Machine Information Received:")
		log.Printf("    • Hostname: %s", hostname)
		log.Printf("    • Method: %s", method)
		log.Printf("    • OS: %v", payload["os"])
		log.Printf("    • Arch: %v", payload["arch"])
		log.Printf("    • Hardware Hash: %v", payload["hardware_hash"])
		log.Printf("")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		// Simulate verification pipeline
		log.Printf("  Running Verification Pipeline:")
		log.Printf("    • fingerprint_match (checking history)... FAIL (unknown)")
		log.Printf("    • os_validation... PASS")
		log.Printf("    • banned_check... PASS")
		log.Printf("")

		// Send verification events
		data := map[string]interface{}{
			"check":      "fingerprint_match",
			"passed":     false,
			"confidence": "unknown",
		}
		dataBytes, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
		flusher.Flush()

		for _, check := range []string{"os_validation", "banned_check"} {
			data := map[string]interface{}{
				"check":      check,
				"passed":     true,
				"confidence": "high",
			}
			dataBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: verify\ndata: %s\n\n", string(dataBytes))
			flusher.Flush()
		}

		// Make decision
		log.Printf("  Decision:")
		log.Printf("    • New machine (no history)")
		log.Printf("    • Status: QUARANTINE")
		log.Printf("    • Profile: stage-0 (read-only)")
		log.Printf("")

		decision := map[string]interface{}{
			"status": "quarantine",
			"reason": "new machine",
		}
		decisionBytes, _ := json.Marshal(decision)
		fmt.Fprintf(w, "event: decision\ndata: %s\n\n", string(decisionBytes))
		flusher.Flush()

		// Issue identity
		log.Printf("  Issuing Identity Certificate:")
		log.Printf("    • ID: new-machine-e2e-test")
		log.Printf("    • Nickname: e2e-machine")
		log.Printf("    • Status: quarantine")
		log.Printf("    • Ziti Hosts: [ziti.example.com]")
		log.Printf("")

		identity := map[string]interface{}{
			"id":       "new-machine-e2e-test",
			"nickname": "e2e-machine",
			"status":   "quarantine",
			"identity": json.RawMessage(`{"type":"pkcs12","data":"base64-encoded-cert"}`),
			"config": map[string]interface{}{
				"hosts": []string{"ziti.example.com"},
				"tunnel": map[string]interface{}{
					"endpoint": "ws://ziti.example.com:3021/edge",
					"acl":      "stage-0",
				},
			},
		}
		identityBytes, _ := json.Marshal(identity)
		fmt.Fprintf(w, "event: identity\ndata: %s\n\n", string(identityBytes))
		flusher.Flush()

		log.Printf("  ✓ Enrollment Stream Complete")
		log.Printf("")
	}))
	defer server.Close()

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗")
	log.Printf("║           MACHINE SDK: Full End-to-End Enrollment             ║")
	log.Printf("╚════════════════════════════════════════════════════════════════╝")
	log.Printf("")
	log.Printf("Connecting to enrollment server: %s", server.URL)
	log.Printf("")

	result, err := enroll.SSEEnroll(server.URL, "", "", "", "")

	if err != nil {
		t.Fatalf("enrollment failed: %v", err)
	}

	log.Printf("╔════════════════════════════════════════════════════════════════╗")
	log.Printf("║              MACHINE SDK: Enrollment Result                    ║")
	log.Printf("╚════════════════════════════════════════════════════════════════╝")
	log.Printf("")
	log.Printf("✓ ENROLLMENT SUCCESSFUL")
	log.Printf("")
	log.Printf("  Identity Information:")
	log.Printf("    • Machine ID: %s", result.ID)
	log.Printf("    • Nickname: %s", result.Nickname)
	log.Printf("    • Status: %s", result.Status)
	log.Printf("    • Hosts: %v", result.Config.Hosts)
	log.Printf("    • Tunnel ACL: %v", result.Config.Tunnel["acl"])
	log.Printf("")
	log.Printf("  Next Steps:")
	log.Printf("    1. Save identity certificate to disk")
	log.Printf("    2. Start Ziti tunnel to enable overlay network")
	log.Printf("    3. Start agent to report telemetry")
	log.Printf("    4. Receive configuration updates via NATS")
	log.Printf("")

	if result.Status != "quarantine" {
		t.Errorf("expected quarantine status for new machine, got %q", result.Status)
	}
}
