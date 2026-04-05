package enroll

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// WebSocketEnroll connects to the WebSocket enrollment endpoint, sends a hello message,
// answers probe questions, and receives the final identity.
// Used for interactive/BrowZer enrollment flows.
func WebSocketEnroll(url, method, session, roleID, secretID string) (*EnrollResult, error) {
	return WebSocketEnrollStream(url, method, session, roleID, secretID, "", func(SSEEvent) {})
}

// WebSocketEnrollStream is like WebSocketEnroll but emits events via callback.
// It implements the WebSocket enrollment protocol as defined in the controller.
func WebSocketEnrollStream(url, method, session, roleID, secretID, profile string, eventFn func(SSEEvent)) (*EnrollResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Connect to WebSocket endpoint
	wsURL := "wss://" + url[len("https://"):] + "/api/ws/enroll"
	if url[:5] == "http:" {
		wsURL = "ws://" + url[len("http://"):] + "/api/ws/enroll"
	}

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send hello message with enrollment method
	hello := map[string]interface{}{
		"type":   "hello",
		"method": method,
	}
	if session != "" {
		hello["session"] = session
	}
	if roleID != "" && secretID != "" {
		hello["role_id"] = roleID
		hello["secret_id"] = secretID
	}
	if profile != "" {
		hello["profile"] = profile
	}

	if err := wsjson.Write(ctx, conn, hello); err != nil {
		return nil, fmt.Errorf("send hello: %w", err)
	}

	if eventFn != nil {
		eventFn(SSEEvent{Kind: "progress", Step: "waiting for probes"})
	}

	// Answer probes (expecting 4: os, hardware, network, system)
	probes := map[string]func() map[string]interface{}{
		"os":       ProbeOS,
		"hardware": ProbeHardware,
		"network":  ProbeNetwork,
		"system":   ProbeSystem,
	}

	for i := 0; i < 4; i++ {
		// Read probe request
		var probe map[string]interface{}
		if err := wsjson.Read(ctx, conn, &probe); err != nil {
			return nil, fmt.Errorf("read probe: %w", err)
		}

		probeType, ok := probe["probe"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid probe message: no probe type")
		}

		if eventFn != nil {
			eventFn(SSEEvent{Kind: "progress", Step: "probing " + probeType})
		}

		// Collect probe data
		var probeData map[string]interface{}
		if fn, exists := probes[probeType]; exists {
			probeData = fn()
		} else {
			return nil, fmt.Errorf("unknown probe type: %s", probeType)
		}

		// Build response with type field
		response := map[string]interface{}{"type": probeType}
		for k, v := range probeData {
			if k != "type" {
				response[k] = v
			}
		}

		if err := wsjson.Write(ctx, conn, response); err != nil {
			return nil, fmt.Errorf("send probe response: %w", err)
		}
	}

	if eventFn != nil {
		eventFn(SSEEvent{Kind: "progress", Step: "waiting for identity"})
	}

	// Read the final identity or error message
	var final map[string]interface{}
	if err := wsjson.Read(ctx, conn, &final); err != nil {
		return nil, fmt.Errorf("read identity: %w", err)
	}

	msgType, _ := final["type"].(string)
	if msgType == "error" {
		reason, _ := final["reason"].(string)
		return nil, fmt.Errorf("server error: %s", reason)
	}

	if msgType == "status" {
		// Rejected
		status, _ := final["status"].(string)
		reason, _ := final["reason"].(string)
		return nil, fmt.Errorf("rejected: %s (%s)", status, reason)
	}

	if msgType != "identity" {
		return nil, fmt.Errorf("unexpected message type: %s", msgType)
	}

	// Parse identity response
	result := &EnrollResult{}
	result.ID, _ = final["id"].(string)
	result.Nickname, _ = final["nickname"].(string)
	result.Status, _ = final["status"].(string)

	// Extract identity certificate
	if identity, ok := final["identity"]; ok {
		result.Identity, _ = json.Marshal(identity)
	}

	// Extract config
	if cfg, ok := final["config"].(map[string]interface{}); ok {
		result.Config.Tunnel, _ = cfg["tunnel"].(map[string]interface{})
		if hosts, ok := cfg["hosts"].([]interface{}); ok {
			for _, h := range hosts {
				if s, ok := h.(string); ok {
					result.Config.Hosts = append(result.Config.Hosts, s)
				}
			}
		}
	}

	if eventFn != nil {
		eventFn(SSEEvent{Kind: "identity", Status: result.Status, Result: result})
	}

	return result, nil
}
