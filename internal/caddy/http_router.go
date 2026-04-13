package caddy

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

// HTTPRouter is a Caddy HTTP handler that implements configuration-first routing
type HTTPRouter struct {
	BaoAddr        string `json:"bao_addr,omitempty"`
	BaoAuthMethod  string `json:"bao_auth_method,omitempty"`   // "cert" or "token"
	BaoCertFile    string `json:"bao_cert_file,omitempty"`     // Path to client cert
	BaoKeyFile     string `json:"bao_key_file,omitempty"`      // Path to client key
	BaoToken       string `json:"bao_token,omitempty"`         // Bearer token
	HoneypotAddr   string `json:"honeypot_addr,omitempty"`     // Fallback addr (e.g., localhost:10443)
	BaoMount       string `json:"bao_mount,omitempty"`         // KV mount (default: "secret")
	BaoNamespace   string `json:"bao_namespace,omitempty"`     // Bao namespace
	DisableLogging bool   `json:"disable_logging,omitempty"`

	logger *zap.Logger
}

// AppConfig represents an app configuration from Bao
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

// CaddyModule returns the Caddy module information
func (h *HTTPRouter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.http_router",
		New: func() caddy.Module { return new(HTTPRouter) },
	}
}

// Provision sets up the handler
func (h *HTTPRouter) Provision(ctx caddy.Context) error {
	h.logger = ctx.Logger(h)

	if h.BaoAddr == "" {
		h.BaoAddr = "https://localhost:8200"
	}
	if h.BaoMount == "" {
		h.BaoMount = "secret"
	}
	if h.HoneypotAddr == "" {
		h.HoneypotAddr = "localhost:10443"
	}
	if h.BaoAuthMethod == "" {
		h.BaoAuthMethod = "cert"
	}

	h.logger.Info("HTTP router provisioned",
		zap.String("bao_addr", h.BaoAddr),
		zap.String("bao_mount", h.BaoMount),
		zap.String("honeypot", h.HoneypotAddr),
	)

	return nil
}

// ServeHTTP implements the HTTP handler interface
func (h *HTTPRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	domain := r.Host
	if idx := strings.Index(domain, ":"); idx >= 0 {
		domain = domain[:idx]
	}

	if !h.DisableLogging {
		h.logger.Debug("routing request",
			zap.String("domain", domain),
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
		)
	}

	// Look up app config in Bao
	config, err := h.lookupAppConfig(domain)
	if err != nil {
		h.logger.Warn("bao lookup failed, using honeypot",
			zap.String("domain", domain),
			zap.Error(err),
		)
		return h.proxyToHoneypot(w, r)
	}

	// No config found
	if config == nil {
		if !h.DisableLogging {
			h.logger.Debug("no config found, using honeypot",
				zap.String("domain", domain),
			)
		}
		return h.proxyToHoneypot(w, r)
	}

	// Check if enabled
	if !config.Enabled {
		h.logger.Info("app disabled, using honeypot",
			zap.String("domain", domain),
			zap.String("service", config.ServiceName),
		)
		return h.proxyToHoneypot(w, r)
	}

	// Check ACL if required
	if config.RequiresAuth {
		allowed := h.checkZitiACL(r, config.ACLTiers)
		if !allowed {
			h.logger.Warn("acl check failed",
				zap.String("domain", domain),
				zap.String("service", config.ServiceName),
				zap.Strings("required_tiers", config.ACLTiers),
			)
			return h.proxyToHoneypot(w, r)
		}
	}

	// Route to backend
	h.logger.Info("routing to backend",
		zap.String("domain", domain),
		zap.String("service", config.ServiceName),
		zap.String("backend", config.Backend),
	)

	return h.proxyToBackend(w, r, config)
}

// lookupAppConfig fetches app config from Bao
func (h *HTTPRouter) lookupAppConfig(domain string) (*AppConfig, error) {
	path := fmt.Sprintf("v1/%s/data/apps/%s/config", h.BaoMount, domain)

	req, err := http.NewRequest("GET", h.BaoAddr+"/"+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set auth
	if h.BaoAuthMethod == "token" && h.BaoToken != "" {
		req.Header.Set("X-Vault-Token", h.BaoToken)
	}

	// Set namespace if provided
	if h.BaoNamespace != "" {
		req.Header.Set("X-Vault-Namespace", h.BaoNamespace)
	}

	client := &http.Client{
		// TODO: Add cert-based auth if BaoAuthMethod == "cert"
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bao request: %w", err)
	}
	defer resp.Body.Close()

	// 404 means no config
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	// Other errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bao returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var baoResp struct {
		Data struct {
			Data AppConfig `json:"data"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&baoResp); err != nil {
		return nil, fmt.Errorf("parse bao response: %w", err)
	}

	return &baoResp.Data.Data, nil
}

// checkZitiACL verifies that the client has required Ziti attributes
func (h *HTTPRouter) checkZitiACL(r *http.Request, requiredTiers []string) bool {
	if len(requiredTiers) == 0 {
		// No ACL defined = deny all
		return false
	}

	// Get client certificate subject (set by Caddy Ziti plugin or mTLS)
	// For now, check X-Ziti-Attributes header (would be set by Caddy plugin)
	attributes := r.Header.Get("X-Ziti-Attributes")
	if attributes == "" {
		// Try alternative header
		attributes = r.Header.Get("X-Ziti-Identity")
		if attributes == "" {
			return false
		}
	}

	// Split attributes
	clientAttrs := strings.Split(attributes, ",")
	for i := range clientAttrs {
		clientAttrs[i] = strings.TrimSpace(clientAttrs[i])
	}

	// Check if any required tier matches any client attribute
	for _, required := range requiredTiers {
		for _, attr := range clientAttrs {
			if attr == required || attr == strings.TrimPrefix(required, "@") {
				return true
			}
		}
	}

	return false
}

// proxyToBackend proxies the request to the configured backend
func (h *HTTPRouter) proxyToBackend(w http.ResponseWriter, r *http.Request, config *AppConfig) error {
	backendURL, err := url.Parse(config.BackendProtocol + "://" + config.Backend)
	if err != nil {
		h.logger.Error("invalid backend URL",
			zap.String("backend", config.Backend),
			zap.Error(err),
		)
		return caddyhttp.Error(http.StatusBadGateway, err)
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			r.Out.Host = backendURL.Host
			r.Out.URL.Scheme = backendURL.Scheme
			r.Out.URL.Host = backendURL.Host

			// Add custom headers
			r.Out.Header.Set("X-App-Name", config.ServiceName)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.logger.Error("backend error",
				zap.String("service", config.ServiceName),
				zap.String("backend", config.Backend),
				zap.Error(err),
			)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}

	proxy.ServeHTTP(w, r)
	return nil
}

// proxyToHoneypot proxies to the 404rd honeypot
func (h *HTTPRouter) proxyToHoneypot(w http.ResponseWriter, r *http.Request) error {
	honeypotURL, err := url.Parse("https://" + h.HoneypotAddr)
	if err != nil {
		h.logger.Error("invalid honeypot URL",
			zap.String("honeypot", h.HoneypotAddr),
			zap.Error(err),
		)
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			r.Out.Host = honeypotURL.Host
			r.Out.URL.Scheme = honeypotURL.Scheme
			r.Out.URL.Host = honeypotURL.Host
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.logger.Error("honeypot error",
				zap.String("honeypot", h.HoneypotAddr),
				zap.Error(err),
			)
			http.Error(w, "Not Found", http.StatusNotFound)
		},
	}

	proxy.ServeHTTP(w, r)
	return nil
}

// UnmarshalCaddyfile parses the Caddyfile directive
func (h *HTTPRouter) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "bao_addr":
				if !d.Args(&h.BaoAddr) {
					return d.ArgErr()
				}
			case "bao_auth_method":
				if !d.Args(&h.BaoAuthMethod) {
					return d.ArgErr()
				}
			case "bao_cert_file":
				if !d.Args(&h.BaoCertFile) {
					return d.ArgErr()
				}
			case "bao_key_file":
				if !d.Args(&h.BaoKeyFile) {
					return d.ArgErr()
				}
			case "bao_token":
				if !d.Args(&h.BaoToken) {
					return d.ArgErr()
				}
			case "bao_mount":
				if !d.Args(&h.BaoMount) {
					return d.ArgErr()
				}
			case "bao_namespace":
				if !d.Args(&h.BaoNamespace) {
					return d.ArgErr()
				}
			case "honeypot_addr":
				if !d.Args(&h.HoneypotAddr) {
					return d.ArgErr()
				}
			case "disable_logging":
				h.DisableLogging = true
			default:
				return d.Errf("unknown directive: %s", d.Val())
			}
		}
	}
	return nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*HTTPRouter)(nil)
	_ caddyhttp.MiddlewareHandler = (*HTTPRouter)(nil)
	_ caddyfile.Unmarshaler       = (*HTTPRouter)(nil)
)
