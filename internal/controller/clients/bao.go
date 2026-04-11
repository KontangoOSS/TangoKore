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

	init, ok := resp["initialized"].(bool)
	if !ok {
		return false, false, fmt.Errorf("invalid initialized field: %T", resp["initialized"])
	}
	sealed, ok = resp["sealed"].(bool)
	if !ok {
		return false, false, fmt.Errorf("invalid sealed field: %T", resp["sealed"])
	}

	return init, sealed, nil
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

	keys, ok := resp["keys"].([]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid keys field: %T", resp["keys"])
	}
	if len(keys) > 0 {
		key, ok := keys[0].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid key in keys array: %T", keys[0])
		}
		unsealKey = key
	}
	token, ok := resp["root_token"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid root_token field: %T", resp["root_token"])
	}
	rootToken = token

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

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	roleID, ok := data["role_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid role_id field: %T", data["role_id"])
	}
	return roleID, nil
}

// CreateAppRoleSecretID creates a new secret ID for an AppRole
func (c *BaoClient) CreateAppRoleSecretID(name string) (string, error) {
	resp, err := c.request("POST", fmt.Sprintf("auth/approle/role/%s/secret-id", name), nil)
	if err != nil {
		return "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	secretID, ok := data["secret_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid secret_id field: %T", data["secret_id"])
	}
	return secretID, nil
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

	auth, ok := resp["auth"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid auth field: %T", resp["auth"])
	}
	token, ok := auth["client_token"].(string)
	if !ok {
		return "", fmt.Errorf("invalid client_token field: %T", auth["client_token"])
	}
	return token, nil
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

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	cert, ok = data["certificate"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid certificate field: %T", data["certificate"])
	}
	key, ok = data["private_key"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid private_key field: %T", data["private_key"])
	}

	return cert, key, nil
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

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	id, ok := data["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid id field: %T", data["id"])
	}
	return id, nil
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

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	id, ok := data["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid id field: %T", data["id"])
	}
	return id, nil
}

// GetMountAccessor retrieves the accessor for a mounted auth method
func (c *BaoClient) GetMountAccessor(authPath string) (string, error) {
	resp, err := c.request("GET", fmt.Sprintf("sys/auth/%s", authPath), nil)
	if err != nil {
		return "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	accessor, ok := data["accessor"].(string)
	if !ok {
		return "", fmt.Errorf("invalid accessor field: %T", data["accessor"])
	}
	return accessor, nil
}

// PKIMount mounts a PKI engine at the given path
func (c *BaoClient) PKIMount(mountPath string) error {
	body := map[string]interface{}{
		"type": "pki",
	}
	_, err := c.request("POST", fmt.Sprintf("sys/mounts/%s", mountPath), body)
	return err
}

// PKIConfigURLs configures the URLs for a PKI mount (issuing CA, CRL distribution)
func (c *BaoClient) PKIConfigURLs(mountPath, issuingURL, crlURL string) error {
	body := map[string]interface{}{
		"issuing_certificates": []string{issuingURL},
		"crl_distribution_points": []string{crlURL},
	}
	_, err := c.request("POST", fmt.Sprintf("%s/config/urls", mountPath), body)
	return err
}

// PKIGenerateRoot generates a self-signed root CA certificate
func (c *BaoClient) PKIGenerateRoot(mountPath, keyType, commonName, ttl string) (certPEM string, err error) {
	body := map[string]interface{}{
		"key_type": keyType,
		"common_name": commonName,
		"ttl": ttl,
	}
	resp, err := c.request("POST", fmt.Sprintf("%s/root/generate/internal", mountPath), body)
	if err != nil {
		return "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	cert, ok := data["certificate"].(string)
	if !ok {
		return "", fmt.Errorf("invalid certificate field: %T", data["certificate"])
	}
	return cert, nil
}

// PKIGenerateIntermediateCSR generates an intermediate CA CSR (with private key exported)
func (c *BaoClient) PKIGenerateIntermediateCSR(mountPath, keyType, commonName string) (csrPEM, keyPEM string, err error) {
	body := map[string]interface{}{
		"key_type": keyType,
		"common_name": commonName,
	}
	resp, err := c.request("POST", fmt.Sprintf("%s/intermediate/generate/exported", mountPath), body)
	if err != nil {
		return "", "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	csr, ok := data["csr"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid csr field: %T", data["csr"])
	}
	key, ok := data["private_key"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid private_key field: %T", data["private_key"])
	}
	return csr, key, nil
}

// PKISignIntermediate signs an intermediate CSR with the root CA
func (c *BaoClient) PKISignIntermediate(rootMount, csr, commonName, ttl string, maxPathLen int) (certPEM string, err error) {
	body := map[string]interface{}{
		"csr": csr,
		"common_name": commonName,
		"ttl": ttl,
		"max_path_length": maxPathLen,
	}
	resp, err := c.request("POST", fmt.Sprintf("%s/root/sign-intermediate", rootMount), body)
	if err != nil {
		return "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	cert, ok := data["certificate"].(string)
	if !ok {
		return "", fmt.Errorf("invalid certificate field: %T", data["certificate"])
	}
	return cert, nil
}

// PKISetSignedIntermediate sets the signed intermediate certificate on the intermediate mount
func (c *BaoClient) PKISetSignedIntermediate(mountPath, certPEM string) error {
	body := map[string]interface{}{
		"certificate": certPEM,
	}
	_, err := c.request("POST", fmt.Sprintf("%s/intermediate/set-signed", mountPath), body)
	return err
}

// PKIIssueCert issues a certificate from a PKI role with optional alt names
func (c *BaoClient) PKIIssueCert(mountPath, roleName, commonName, ttl string, altNames []string) (cert, key, ca string, err error) {
	body := map[string]interface{}{
		"common_name": commonName,
		"ttl": ttl,
	}
	if len(altNames) > 0 {
		body["alt_names"] = altNames
	}
	resp, err := c.request("POST", fmt.Sprintf("%s/issue/%s", mountPath, roleName), body)
	if err != nil {
		return "", "", "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", "", "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	cert, ok = data["certificate"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("invalid certificate field: %T", data["certificate"])
	}
	key, ok = data["private_key"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("invalid private_key field: %T", data["private_key"])
	}
	ca, ok = data["ca_chain"].(string)
	if !ok {
		// ca_chain may not always be present, use certificate_chain or ca
		if chain, ok := data["ca_chain"]; ok {
			ca, _ = chain.(string)
		}
	}
	return cert, key, ca, nil
}

// EnableAppRole enables the AppRole auth method (returns error if already enabled, which is OK)
func (c *BaoClient) EnableAppRole() error {
	return c.EnableAuth("approle", "approle")
}

// CreateAppRoleRole creates an AppRole with policies and TTL
func (c *BaoClient) CreateAppRoleRole(roleName string, policies []string, ttl string) error {
	body := map[string]interface{}{
		"token_ttl": ttl,
		"token_max_ttl": "24h",
		"policies": policies,
	}
	_, err := c.request("POST", fmt.Sprintf("auth/approle/role/%s", roleName), body)
	return err
}

// CreateAppRoleSecret creates a new secret ID for an AppRole role
func (c *BaoClient) CreateAppRoleSecret(roleName string) (string, error) {
	resp, err := c.request("POST", fmt.Sprintf("auth/approle/role/%s/secret-id", roleName), nil)
	if err != nil {
		return "", err
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid data field: %T", resp["data"])
	}
	secretID, ok := data["secret_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid secret_id field: %T", data["secret_id"])
	}
	return secretID, nil
}

// EnableCertAuth enables the cert auth method at a specific path
func (c *BaoClient) EnableCertAuth(mountPath string) error {
	return c.EnableAuth(mountPath, "cert")
}

// CreateCertAuthRole creates a certificate auth role that maps certificates to policies
func (c *BaoClient) CreateCertAuthRole(mountPath, roleName, certPEM string, policies []string) error {
	body := map[string]interface{}{
		"certificate": certPEM,
		"display_name": roleName,
		"ttl": "24h",
		"max_ttl": "24h",
		"policies": policies,
	}
	_, err := c.request("POST", fmt.Sprintf("auth/%s/certs/%s", mountPath, roleName), body)
	return err
}
