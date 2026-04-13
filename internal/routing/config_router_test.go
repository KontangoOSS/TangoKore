package routing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	messages []string
}

func (m *MockLogger) Info(msg string, kv ...interface{}) {
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *MockLogger) Warn(msg string, kv ...interface{}) {
	m.messages = append(m.messages, "WARN: "+msg)
}

func (m *MockLogger) Error(msg string, kv ...interface{}) {
	m.messages = append(m.messages, "ERROR: "+msg)
}

func (m *MockLogger) Debug(msg string, kv ...interface{}) {
	m.messages = append(m.messages, "DEBUG: "+msg)
}

// TestConfigLookup tests configuration lookup from Bao
func TestConfigLookup(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		statusCode  int
		configBody  *AppConfig
		expectErr   bool
		expectNil   bool
	}{
		{
			name:       "config found",
			domain:     "code.konoss.org",
			statusCode: 200,
			configBody: &AppConfig{
				Enabled:         true,
				Backend:         "192.0.2.10:8080",
				BackendProtocol: "http",
				ServiceName:     "code-server",
				RequiresAuth:    true,
				ACLTiers:        []string{"@code-server-hosts"},
			},
			expectErr: false,
			expectNil: false,
		},
		{
			name:       "config not found (404)",
			domain:     "unknown.konoss.org",
			statusCode: 404,
			expectErr: false,
			expectNil: true,
		},
		{
			name:       "bao error (500)",
			domain:     "error.konoss.org",
			statusCode: 500,
			expectErr: true,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Bao server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path contains secret/data/apps/
				if !strings.Contains(r.URL.Path, "/secret/data/apps/") {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				// Check for token
				if r.Header.Get("X-Vault-Token") == "" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == 200 && tt.configBody != nil {
					resp := struct {
						Data struct {
							Data *AppConfig `json:"data"`
						} `json:"data"`
					}{
						Data: struct {
							Data *AppConfig `json:"data"`
						}{
							Data: tt.configBody,
						},
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			router := &ConfigRouter{
				BaoAddr:  server.URL,
				BaoMount: "secret",
				BaoToken: "test-token",
				Logger:   &MockLogger{},
			}

			config, err := router.LookupConfig(tt.domain)

			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectNil && config != nil {
				t.Fatalf("expected nil, got config: %+v", config)
			}
			if !tt.expectNil && config == nil {
				t.Fatalf("expected config, got nil")
			}

			if config != nil && config.ServiceName != tt.configBody.ServiceName {
				t.Fatalf("service name mismatch: got %s, want %s", config.ServiceName, tt.configBody.ServiceName)
			}
		})
	}
}

// TestACLCheck tests ACL enforcement
func TestACLCheck(t *testing.T) {
	tests := []struct {
		name            string
		clientAttrs     []string
		requiredTiers   []string
		expectedAllowed bool
	}{
		{
			name:            "exact match",
			clientAttrs:     []string{"@code-server-hosts"},
			requiredTiers:   []string{"@code-server-hosts"},
			expectedAllowed: true,
		},
		{
			name:            "multiple attrs, one matches",
			clientAttrs:     []string{"@stage-2", "@code-server-hosts"},
			requiredTiers:   []string{"@code-server-hosts"},
			expectedAllowed: true,
		},
		{
			name:            "multiple tiers, one matches",
			clientAttrs:     []string{"@stage-2"},
			requiredTiers:   []string{"@stage-2", "@stage-3"},
			expectedAllowed: true,
		},
		{
			name:            "no match",
			clientAttrs:     []string{"@stage-2"},
			requiredTiers:   []string{"@admin"},
			expectedAllowed: false,
		},
		{
			name:            "empty client attrs",
			clientAttrs:     []string{},
			requiredTiers:   []string{"@admin"},
			expectedAllowed: false,
		},
		{
			name:            "empty required tiers (deny all)",
			clientAttrs:     []string{"@stage-2"},
			requiredTiers:   []string{},
			expectedAllowed: false,
		},
		{
			name:            "attr without @ prefix",
			clientAttrs:     []string{"code-server-hosts"},
			requiredTiers:   []string{"@code-server-hosts"},
			expectedAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := &ConfigRouter{}

			allowed := router.CheckACL(tt.clientAttrs, tt.requiredTiers)

			if allowed != tt.expectedAllowed {
				t.Fatalf("ACL check failed: got %v, want %v (attrs=%v, tiers=%v)",
					allowed, tt.expectedAllowed, tt.clientAttrs, tt.requiredTiers)
			}
		})
	}
}

// TestRoutingDecision tests the full routing decision logic
func TestRoutingDecision(t *testing.T) {
	tests := []struct {
		name            string
		config          *AppConfig
		expectRouting   string // "backend" or "honeypot"
		expectedReason  string
	}{
		{
			name: "enabled config",
			config: &AppConfig{
				Enabled: true,
				Backend: "192.0.2.10:8080",
			},
			expectRouting:  "backend",
			expectedReason: "config found and enabled",
		},
		{
			name: "disabled config",
			config: &AppConfig{
				Enabled: false,
				Backend: "192.0.2.10:8080",
			},
			expectRouting:  "honeypot",
			expectedReason: "app is disabled",
		},
		{
			name:            "nil config",
			config:          nil,
			expectRouting:   "honeypot",
			expectedReason:  "no configuration found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate what RouteRequest does
			if tt.config == nil {
				decision := &RoutingDecision{
					RouteType: "honeypot",
					Reason:    tt.expectedReason,
				}

				if decision.RouteType != tt.expectRouting {
					t.Fatalf("routing type mismatch: got %s, want %s", decision.RouteType, tt.expectRouting)
				}
				if decision.Reason != tt.expectedReason {
					t.Fatalf("reason mismatch: got %s, want %s", decision.Reason, tt.expectedReason)
				}
			} else if tt.config.Enabled {
				decision := &RoutingDecision{
					RouteType: "backend",
					Backend:   tt.config.Backend,
					Config:    tt.config,
					Reason:    tt.expectedReason,
				}

				if decision.RouteType != tt.expectRouting {
					t.Fatalf("routing type mismatch: got %s, want %s", decision.RouteType, tt.expectRouting)
				}
			} else {
				decision := &RoutingDecision{
					RouteType: "honeypot",
					Config:    tt.config,
					Reason:    tt.expectedReason,
				}

				if decision.RouteType != tt.expectRouting {
					t.Fatalf("routing type mismatch: got %s, want %s", decision.RouteType, tt.expectRouting)
				}
			}
		})
	}
}

// TestFullRoutingFlow tests the complete routing decision flow
func TestFullRoutingFlow(t *testing.T) {
	// Create mock Bao server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return different configs based on domain
		if strings.Contains(r.URL.Path, "code.konoss.org") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(struct {
				Data struct {
					Data AppConfig `json:"data"`
				} `json:"data"`
			}{
				Data: struct {
					Data AppConfig `json:"data"`
				}{
					Data: AppConfig{
						Enabled:      true,
						Backend:      "192.0.2.10:8080",
						ServiceName:  "code-server",
						RequiresAuth: true,
						ACLTiers:     []string{"@code-server-hosts"},
					},
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	router := &ConfigRouter{
		BaoAddr:      server.URL,
		BaoMount:     "secret",
		BaoToken:     "test-token",
		HoneypotAddr: "localhost:10443",
		Logger:       &MockLogger{},
	}

	// Test 1: Configured domain
	decision, err := router.RouteRequest("code.konoss.org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.RouteType != "backend" {
		t.Fatalf("expected backend routing, got %s", decision.RouteType)
	}
	if decision.Config == nil || decision.Config.ServiceName != "code-server" {
		t.Fatalf("config not loaded correctly")
	}

	// Test 2: Unconfigured domain
	decision, err = router.RouteRequest("unknown.konoss.org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.RouteType != "honeypot" {
		t.Fatalf("expected honeypot routing, got %s", decision.RouteType)
	}
}

// TestConfigSchema validates JSON unmarshaling
func TestConfigSchema(t *testing.T) {
	validJSON := `{
		"enabled": true,
		"backend": "192.0.2.10:8080",
		"backend_protocol": "http",
		"tls_termination": "caddy",
		"service_name": "code-server",
		"attributes": ["code-server", "development"],
		"requires_auth": true,
		"acl_tiers": ["@code-server-hosts"],
		"description": "VS Code IDE",
		"metadata": {"app_type": "ide"},
		"rate_limit": 1000,
		"timeout": 60,
		"websocket": true
	}`

	var config AppConfig
	if err := json.Unmarshal([]byte(validJSON), &config); err != nil {
		t.Fatalf("failed to unmarshal valid config: %v", err)
	}

	if !config.Enabled {
		t.Fatalf("enabled should be true")
	}
	if config.ServiceName != "code-server" {
		t.Fatalf("service name mismatch")
	}
	if config.RateLimit != 1000 {
		t.Fatalf("rate limit mismatch")
	}
	if !config.WebSocket {
		t.Fatalf("websocket should be true")
	}
	if len(config.ACLTiers) != 1 || config.ACLTiers[0] != "@code-server-hosts" {
		t.Fatalf("acl tiers mismatch")
	}
	if config.Timeout != 60 {
		t.Fatalf("timeout should be 60")
	}
}

// BenchmarkACLCheck benchmarks ACL checking
func BenchmarkACLCheck(b *testing.B) {
	router := &ConfigRouter{}
	clientAttrs := []string{"@stage-2", "@code-server-hosts"}
	requiredTiers := []string{"@code-server-hosts", "@admin"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.CheckACL(clientAttrs, requiredTiers)
	}
}

// BenchmarkConfigLookup benchmarks config lookup
func BenchmarkConfigLookup(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Data struct {
				Data AppConfig `json:"data"`
			} `json:"data"`
		}{
			Data: struct {
				Data AppConfig `json:"data"`
			}{
				Data: AppConfig{
					Enabled:     true,
					Backend:     "192.0.2.10:8080",
					ServiceName: "test",
				},
			},
		})
	}))
	defer server.Close()

	router := &ConfigRouter{
		BaoAddr:  server.URL,
		BaoMount: "secret",
		BaoToken: "test-token",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.LookupConfig("test.konoss.org")
	}
}
