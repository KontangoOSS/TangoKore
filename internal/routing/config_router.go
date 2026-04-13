package routing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// ConfigRouter implements configuration-first routing
// It looks up app configs in Bao KV and routes accordingly
type ConfigRouter struct {
	// Bao configuration
	BaoAddr      string // e.g., "https://localhost:8200"
	BaoMount     string // e.g., "secret"
	BaoToken     string // Auth token
	BaoNamespace string // Optional Bao namespace

	// Fallback addresses
	HoneypotAddr  string // e.g., "localhost:10443" (404rd)
	DefaultBackend string // Fallback if no config

	// Logging
	Logger Logger
}

// Logger is a simple logging interface
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// AppConfig represents an application configuration from Bao
type AppConfig struct {
	Enabled            bool                   `json:"enabled"`
	Backend            string                 `json:"backend"`
	BackendProtocol    string                 `json:"backend_protocol"`
	TLSTermination     string                 `json:"tls_termination"`
	ServiceName        string                 `json:"service_name"`
	Attributes         []string               `json:"attributes"`
	RequiresAuth       bool                   `json:"requires_auth"`
	ACLTiers           []string               `json:"acl_tiers"`
	Description        string                 `json:"description"`
	Metadata           map[string]interface{} `json:"metadata"`
	RateLimit          int                    `json:"rate_limit"`
	Timeout            int                    `json:"timeout"`
	WebSocket          bool                   `json:"websocket"`
}

// RoutingDecision represents the decision made by the router
type RoutingDecision struct {
	RouteType  string     // "backend", "honeypot", "error"
	Backend    string     // Target backend address
	Config     *AppConfig // App config (if routed to backend)
	Reason     string     // Why this decision was made
}

// RouteRequest makes a routing decision for the given domain
// Returns the routing decision and an error (if routing failed)
func (cr *ConfigRouter) RouteRequest(domain string) (*RoutingDecision, error) {
	cr.log("info", fmt.Sprintf("routing request for domain: %s", domain))

	// 1. Look up config in Bao
	config, err := cr.LookupConfig(domain)
	if err != nil {
		cr.log("warn", fmt.Sprintf("bao lookup error for %s: %v", domain, err))
		return &RoutingDecision{
			RouteType: "honeypot",
			Reason:    fmt.Sprintf("bao lookup error: %v", err),
		}, nil // Error is recoverable - fall back to honeypot
	}

	// 2. No config found
	if config == nil {
		cr.log("debug", fmt.Sprintf("no config found for %s, using honeypot", domain))
		return &RoutingDecision{
			RouteType: "honeypot",
			Reason:    "no configuration found",
		}, nil
	}

	// 3. Config found - check if enabled
	if !config.Enabled {
		cr.log("info", fmt.Sprintf("app disabled for %s, using honeypot", domain))
		return &RoutingDecision{
			RouteType: "honeypot",
			Config:    config,
			Reason:    "app is disabled",
		}, nil
	}

	// 4. Route to configured backend
	cr.log("info", fmt.Sprintf("routing %s to backend: %s", domain, config.Backend))
	return &RoutingDecision{
		RouteType: "backend",
		Backend:   config.Backend,
		Config:    config,
		Reason:    "config found and enabled",
	}, nil
}

// CheckACL verifies that a client has required Ziti ACL tiers
// Returns true if client attributes match any required tier
func (cr *ConfigRouter) CheckACL(clientAttributes []string, requiredTiers []string) bool {
	if len(requiredTiers) == 0 {
		// No ACL defined = deny all
		return false
	}

	if len(clientAttributes) == 0 {
		return false
	}

	// Check if any required tier matches any client attribute
	for _, required := range requiredTiers {
		for _, attr := range clientAttributes {
			if attr == required || attr == strings.TrimPrefix(required, "@") {
				return true
			}
		}
	}

	return false
}

// LookupConfig fetches app configuration from Bao KV
// Returns nil if no config found (404), error if Bao is unreachable
func (cr *ConfigRouter) LookupConfig(domain string) (*AppConfig, error) {
	path := fmt.Sprintf("%s/v1/%s/data/apps/%s/config", cr.BaoAddr, cr.BaoMount, domain)

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add authentication
	if cr.BaoToken != "" {
		req.Header.Set("X-Vault-Token", cr.BaoToken)
	}

	// Add namespace if provided
	if cr.BaoNamespace != "" {
		req.Header.Set("X-Vault-Namespace", cr.BaoNamespace)
	}

	// Make request (insecure for testing)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bao request failed: %w", err)
	}
	defer resp.Body.Close()

	// 404 means no config exists
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	// Other errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bao returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse Bao response format: {data: {data: <config>}}
	var baoResp struct {
		Data struct {
			Data AppConfig `json:"data"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&baoResp); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &baoResp.Data.Data, nil
}

// ProxyToBackend creates a reverse proxy to the backend
func (cr *ConfigRouter) ProxyToBackend(w http.ResponseWriter, r *http.Request, backendAddr string) error {
	// Parse backend address
	backendURL, err := url.Parse("http://" + backendAddr)
	if err != nil {
		cr.log("error", fmt.Sprintf("invalid backend address %s: %v", backendAddr, err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return err
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			r.Out.Host = backendURL.Host
			r.Out.URL.Scheme = backendURL.Scheme
			r.Out.URL.Host = backendURL.Host
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			cr.log("error", fmt.Sprintf("backend error: %v", err))
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}

	proxy.ServeHTTP(w, r)
	return nil
}

// ProxyToHoneypot creates a reverse proxy to the honeypot
func (cr *ConfigRouter) ProxyToHoneypot(w http.ResponseWriter, r *http.Request) error {
	honeypotURL, err := url.Parse("https://" + cr.HoneypotAddr)
	if err != nil {
		cr.log("error", fmt.Sprintf("invalid honeypot address %s: %v", cr.HoneypotAddr, err))
		http.Error(w, "Not Found", http.StatusNotFound)
		return err
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			r.Out.Host = honeypotURL.Host
			r.Out.URL.Scheme = honeypotURL.Scheme
			r.Out.URL.Host = honeypotURL.Host
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			cr.log("error", fmt.Sprintf("honeypot error: %v", err))
			http.Error(w, "Not Found", http.StatusNotFound)
		},
	}

	proxy.ServeHTTP(w, r)
	return nil
}

// log is a helper for logging
func (cr *ConfigRouter) log(level string, msg string) {
	if cr.Logger == nil {
		return
	}
	switch level {
	case "info":
		cr.Logger.Info(msg)
	case "warn":
		cr.Logger.Warn(msg)
	case "error":
		cr.Logger.Error(msg)
	case "debug":
		cr.Logger.Debug(msg)
	}
}
