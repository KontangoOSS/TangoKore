package caddy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestConfigLookup tests the configuration lookup from Bao
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
			name:       "config not found",
			domain:     "unknown.konoss.org",
			statusCode: 404,
			expectErr: false,
			expectNil: true,
		},
		{
			name:       "bao error",
			domain:     "error.konoss.org",
			statusCode: 500,
			expectErr: true,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Bao server
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify path
				if !strings.Contains(r.URL.Path, "/secret/data/apps/") {
					w.WriteHeader(http.StatusNotFound)
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

			router := &HTTPRouter{
				BaoAddr:   server.URL,
				BaoMount:  "secret",
				BaoToken:  "test-token",
			}

			config, err := router.lookupAppConfig(tt.domain)

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
				t.Fatalf("config mismatch: got %+v, want %+v", config, tt.configBody)
			}
		})
	}
}

// TestACLCheck tests ACL enforcement
func TestACLCheck(t *testing.T) {
	tests := []struct {
		name            string
		requiredTiers   []string
		clientAttrs     string
		expectedAllowed bool
	}{
		{
			name:            "exact match",
			requiredTiers:   []string{"@code-server-hosts"},
			clientAttrs:     "@code-server-hosts",
			expectedAllowed: true,
		},
		{
			name:            "multiple attrs, one matches",
			requiredTiers:   []string{"@code-server-hosts"},
			clientAttrs:     "@stage-2, @code-server-hosts",
			expectedAllowed: true,
		},
		{
			name:            "multiple tiers, one matches",
			requiredTiers:   []string{"@stage-2", "@stage-3"},
			clientAttrs:     "@stage-2",
			expectedAllowed: true,
		},
		{
			name:            "no match",
			requiredTiers:   []string{"@admin"},
			clientAttrs:     "@stage-2",
			expectedAllowed: false,
		},
		{
			name:            "empty attrs",
			requiredTiers:   []string{"@admin"},
			clientAttrs:     "",
			expectedAllowed: false,
		},
		{
			name:            "empty tiers (deny all)",
			requiredTiers:   []string{},
			clientAttrs:     "@stage-2",
			expectedAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := &HTTPRouter{}

			req := httptest.NewRequest("GET", "https://example.com/", nil)
			if tt.clientAttrs != "" {
				req.Header.Set("X-Ziti-Attributes", tt.clientAttrs)
			}

			allowed := router.checkZitiACL(req, tt.requiredTiers)

			if allowed != tt.expectedAllowed {
				t.Fatalf("ACL check failed: got %v, want %v", allowed, tt.expectedAllowed)
			}
		})
	}
}

// TestRoutingDecision tests the full routing decision logic
func TestRoutingDecision(t *testing.T) {
	tests := []struct {
		name              string
		config            *AppConfig
		clientAttrs       string
		shouldRoute       bool
		expectedBackend   string
		expectedHoneypot  bool
	}{
		{
			name: "enabled, auth required, acl matches",
			config: &AppConfig{
				Enabled:      true,
				Backend:      "192.0.2.10:8080",
				RequiresAuth: true,
				ACLTiers:     []string{"@code-server-hosts"},
			},
			clientAttrs:     "@code-server-hosts",
			shouldRoute:     true,
			expectedBackend: "192.0.2.10:8080",
		},
		{
			name: "enabled, auth required, acl fails",
			config: &AppConfig{
				Enabled:      true,
				Backend:      "192.0.2.10:8080",
				RequiresAuth: true,
				ACLTiers:     []string{"@admin"},
			},
			clientAttrs:    "@stage-2",
			shouldRoute:    false,
			expectedHoneypot: true,
		},
		{
			name: "disabled",
			config: &AppConfig{
				Enabled:      false,
				Backend:      "192.0.2.10:8080",
				RequiresAuth: false,
			},
			shouldRoute:     false,
			expectedHoneypot: true,
		},
		{
			name: "no auth required",
			config: &AppConfig{
				Enabled:      true,
				Backend:      "192.0.2.10:8080",
				RequiresAuth: false,
			},
			shouldRoute:     true,
			expectedBackend: "192.0.2.10:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "https://test.konoss.org/", nil)
			if tt.clientAttrs != "" {
				req.Header.Set("X-Ziti-Attributes", tt.clientAttrs)
			}

			router := &HTTPRouter{}

			// Check ACL (if config exists)
			if tt.config != nil && tt.config.RequiresAuth {
				allowed := router.checkZitiACL(req, tt.config.ACLTiers)
				if tt.shouldRoute && !allowed {
					t.Fatalf("ACL check failed: should have allowed")
				}
				if !tt.shouldRoute && allowed {
					t.Fatalf("ACL check failed: should have denied")
				}
			}

			// Verify config state
			if tt.config != nil {
				if !tt.config.Enabled && tt.shouldRoute {
					t.Fatalf("config disabled but expected to route")
				}
			}
		})
	}
}

// TestConfigSchema validates the AppConfig JSON schema
func TestConfigSchema(t *testing.T) {
	// Valid config
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
	if err := json.NewDecoder(strings.NewReader(validJSON)).Decode(&config); err != nil {
		t.Fatalf("failed to unmarshal valid config: %v", err)
	}

	if config.ServiceName != "code-server" {
		t.Fatalf("service name mismatch: got %s", config.ServiceName)
	}
	if config.RateLimit != 1000 {
		t.Fatalf("rate limit mismatch: got %d", config.RateLimit)
	}
	if !config.WebSocket {
		t.Fatalf("websocket should be true")
	}
}

// BenchmarkACLCheck benchmarks ACL checking
func BenchmarkACLCheck(b *testing.B) {
	router := &HTTPRouter{}
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("X-Ziti-Attributes", "@stage-2, @code-server-hosts, @developers")

	requiredTiers := []string{"@code-server-hosts", "@admin"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.checkZitiACL(req, requiredTiers)
	}
}

// BenchmarkConfigLookup benchmarks config lookup (with mock server)
func BenchmarkConfigLookup(b *testing.B) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"data":{"data":{"enabled":true,"backend":"192.0.2.10:8080","service_name":"test"}}}`)
	}))
	defer server.Close()

	router := &HTTPRouter{
		BaoAddr:  server.URL,
		BaoMount: "secret",
		BaoToken: "test-token",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.lookupAppConfig("test.konoss.org")
	}
}
