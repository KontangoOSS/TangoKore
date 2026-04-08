package clients

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

// BaoClient wraps the Bao vault API
type BaoClient struct {
	Addr      string // "https://127.0.0.1:8200"
	Token     string
	CACert    string // path to ca-bundle.pem; empty = skip verify
	client    *http.Client
	Insecure  bool
}

// NewBaoClient creates a new Bao client
func NewBaoClient(addr, token, caCert string) (*BaoClient, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Default for test mode
	}

	// If CA cert is provided, load it (production mode)
	if caCert != "" {
		// TODO: load CA cert
		tlsConfig.InsecureSkipVerify = false
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &BaoClient{
		Addr:     addr,
		Token:    token,
		CACert:   caCert,
		client:   httpClient,
		Insecure: true,
	}, nil
}

// WithToken returns a new client with a different token
func (c *BaoClient) WithToken(token string) *BaoClient {
	newC := *c
	newC.Token = token
	return &newC
}

// request makes an HTTP request to Bao
func (c *BaoClient) request(method, endpoint string, body interface{}) (map[string]interface{}, error) {
	u, err := url.Parse(c.Addr)
	if err != nil {
		return nil, fmt.Errorf("parse addr: %w", err)
	}
	u.Path = path.Join(u.Path, "v1", endpoint)

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("X-Vault-Token", c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return result, nil
}

// SealStatus returns seal status of Bao
func (c *BaoClient) SealStatus() (initialized, sealed bool, err error) {
	resp, err := c.request("GET", "sys/seal-status", nil)
	if err != nil {
		return false, false, err
	}

	initialized = resp["initialized"].(bool)
	sealed = resp["sealed"].(bool)

	return initialized, sealed, nil
}

// Init initializes Bao with given shares/threshold
func (c *BaoClient) Init(shares, threshold int) (unsealKey, rootToken string, err error) {
	body := map[string]interface{}{
		"secret_shares":    shares,
		"secret_threshold": threshold,
	}

	resp, err := c.request("POST", "sys/init", body)
	if err != nil {
		return "", "", err
	}

	keys := resp["keys"].([]interface{})
	if len(keys) > 0 {
		unsealKey = keys[0].(string)
	}
	rootToken = resp["root_token"].(string)

	return unsealKey, rootToken, nil
}

// Unseal unseals Bao with a key
func (c *BaoClient) Unseal(key string) error {
	body := map[string]interface{}{
		"key": key,
	}

	_, err := c.request("POST", "sys/unseal", body)
	return err
}

// EnableEngine enables a secrets engine at the given path
func (c *BaoClient) EnableEngine(path, engineType string) error {
	body := map[string]interface{}{
		"type": engineType,
	}

	_, err := c.request("POST", fmt.Sprintf("sys/mounts/%s", path), body)
	return err
}

// EnableAuth enables an auth method at the given path
func (c *BaoClient) EnableAuth(path, authType string) error {
	body := map[string]interface{}{
		"type": authType,
	}

	_, err := c.request("POST", fmt.Sprintf("sys/auth/%s", path), body)
	return err
}

// KVPut stores a secret in KV v2
func (c *BaoClient) KVPut(mount, path string, data map[string]interface{}) error {
	body := map[string]interface{}{
		"data": data,
	}

	_, err := c.request("POST", fmt.Sprintf("%s/data/%s", mount, path), body)
	return err
}

// KVGet retrieves a secret from KV v2
func (c *BaoClient) KVGet(mount, path string) (map[string]interface{}, error) {
	resp, err := c.request("GET", fmt.Sprintf("%s/data/%s", mount, path), nil)
	if err != nil {
		return nil, err
	}

	// KV v2 wraps data in response.data.data
	if data, ok := resp["data"].(map[string]interface{}); ok {
		if innerData, ok := data["data"].(map[string]interface{}); ok {
			return innerData, nil
		}
	}

	return nil, fmt.Errorf("unexpected KV response structure")
}

// CreatePolicy creates a policy in Bao
func (c *BaoClient) CreatePolicy(name, hcl string) error {
	body := map[string]interface{}{
		"policy": hcl,
	}

	_, err := c.request("POST", fmt.Sprintf("sys/policies/acl/%s", name), body)
	return err
}

// CreateAppRole creates an AppRole
func (c *BaoClient) CreateAppRole(name string, policies []string, tokenTTL string) error {
	body := map[string]interface{}{
		"token_ttl":     tokenTTL,
		"token_max_ttl": "24h",
		"policies":      policies,
	}

	_, err := c.request("POST", fmt.Sprintf("auth/approle/role/%s", name), body)
	return err
}

// GetAppRoleID retrieves the role ID for an AppRole
func (c *BaoClient) GetAppRoleID(name string) (string, error) {
	resp, err := c.request("GET", fmt.Sprintf("auth/approle/role/%s/role-id", name), nil)
	if err != nil {
		return "", err
	}

	data := resp["data"].(map[string]interface{})
	return data["role_id"].(string), nil
}

// CreateAppRoleSecretID creates a new secret ID for an AppRole
func (c *BaoClient) CreateAppRoleSecretID(name string) (string, error) {
	resp, err := c.request("POST", fmt.Sprintf("auth/approle/role/%s/secret-id", name), nil)
	if err != nil {
		return "", err
	}

	data := resp["data"].(map[string]interface{})
	return data["secret_id"].(string), nil
}

// AppRoleLogin logs in with AppRole credentials
func (c *BaoClient) AppRoleLogin(roleID, secretID string) (string, error) {
	body := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	resp, err := c.request("POST", "auth/approle/login", body)
	if err != nil {
		return "", err
	}

	auth := resp["auth"].(map[string]interface{})
	return auth["client_token"].(string), nil
}

// CreatePKIRole creates a PKI role in the pki mount
func (c *BaoClient) CreatePKIRole(roleName string, allowedDomains []string, allowSubdomains bool, maxTTL string) error {
	body := map[string]interface{}{
		"allowed_domains":   allowedDomains,
		"allow_subdomains":  allowSubdomains,
		"max_ttl":           maxTTL,
		"key_type":          "rsa",
		"key_bits":          2048,
		"require_cn":        false,
		"generate_lease":    true,
		"server_flag":       true,
		"client_flag":       true,
	}

	_, err := c.request("POST", fmt.Sprintf("pki/roles/%s", roleName), body)
	return err
}

// IssueCert issues a certificate from a PKI role
func (c *BaoClient) IssueCert(roleName, commonName string, ttl string) (cert, key string, err error) {
	body := map[string]interface{}{
		"common_name": commonName,
		"ttl":         ttl,
	}

	resp, err := c.request("POST", fmt.Sprintf("pki/issue/%s", roleName), body)
	if err != nil {
		return "", "", err
	}

	data := resp["data"].(map[string]interface{})
	cert = data["certificate"].(string)
	key = data["private_key"].(string)

	return cert, key, nil
}

// EnableCertAuth enables certificate-based authentication
func (c *BaoClient) EnableCertAuth(path string) error {
	return c.EnableAuth(path, "cert")
}

// CreateCertRole creates a certificate auth role that maps cert CN patterns to identities
// For example: role "lab-devices" could match CN pattern "*.lab.example.com" -> identity "lab-stage"
func (c *BaoClient) CreateCertRole(name string, certCNPattern string, identityName string, policies []string) error {
	body := map[string]interface{}{
		"certificate": certCNPattern,
		"display_name": name,
		"ttl": "24h",
		"max_ttl": "24h",
		"policies": policies,
		// These fields would map the cert to an entity/identity
		"bind_cidrs": []string{}, // allow all
		"ocsp_fail_open": true,   // allow offline OCSP
	}

	_, err := c.request("POST", fmt.Sprintf("auth/cert/certs/%s", name), body)
	return err
}

// CreateEntityAlias creates an alias linking a certificate CN to a Bao entity (identity)
// This allows certificate groups (e.g., all *.lab.{domain} certs) to authenticate as a single entity
func (c *BaoClient) CreateEntityAlias(authPath string, cnPattern string, entityID string) (string, error) {
	body := map[string]interface{}{
		"name":           cnPattern,
		"canonical_id":   entityID,
		"mount_accessor": authPath, // Will need to lookup mount accessor
	}

	resp, err := c.request("POST", "identity/entity-alias", body)
	if err != nil {
		return "", err
	}

	data := resp["data"].(map[string]interface{})
	return data["id"].(string), nil
}

// CreateEntity creates a named entity (identity) in Bao
// This represents a group of devices at the same stage
func (c *BaoClient) CreateEntity(name string, policies []string, metadata map[string]string) (string, error) {
	body := map[string]interface{}{
		"name":     name,
		"policies": policies,
		"metadata": metadata,
	}

	resp, err := c.request("POST", "identity/entity", body)
	if err != nil {
		return "", err
	}

	data := resp["data"].(map[string]interface{})
	return data["id"].(string), nil
}

// GetMountAccessor retrieves the accessor for a mounted auth method
func (c *BaoClient) GetMountAccessor(authPath string) (string, error) {
	resp, err := c.request("GET", fmt.Sprintf("sys/auth/%s", authPath), nil)
	if err != nil {
		return "", err
	}

	data := resp["data"].(map[string]interface{})
	accessor := data["accessor"].(string)
	return accessor, nil
}
