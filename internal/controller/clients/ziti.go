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

// ZitiClient is a Ziti management REST client
type ZitiClient struct {
	Addr     string // "localhost:1280"
	User     string
	Pass     string
	Token    string
	client   *http.Client
	Insecure bool
}

// NewZitiClient creates a new Ziti client
func NewZitiClient(addr, user, pass string) (*ZitiClient, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &ZitiClient{
		Addr:     addr,
		User:     user,
		Pass:     pass,
		client:   httpClient,
		Insecure: true,
	}, nil
}

// request makes an HTTP request to Ziti
func (c *ZitiClient) request(method, endpoint string, body interface{}) (map[string]interface{}, error) {
	u := url.URL{
		Scheme: "https",
		Host:   c.Addr,
		Path:   path.Join("/edge/management/v1", endpoint),
	}

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

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
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

// Authenticate logs in and stores the token
func (c *ZitiClient) Authenticate() (token string, err error) {
	body := map[string]interface{}{
		"username": c.User,
		"password": c.Pass,
	}

	resp, err := c.request("POST", "authenticate", body)
	if err != nil {
		return "", err
	}

	data := resp["data"].(map[string]interface{})
	c.Token = data["token"].(string)
	return c.Token, nil
}

// FindByName finds an entity by type and name, returns ID
func (c *ZitiClient) FindByName(token, entityType, name string) (string, error) {
	filter := url.QueryEscape(fmt.Sprintf(`name="%s"`, name))
	endpoint := fmt.Sprintf("%ss?filter=%s", entityType, filter)

	resp, err := c.request("GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	data := resp["data"].([]interface{})
	if len(data) == 0 {
		return "", fmt.Errorf("not found: %s %s", entityType, name)
	}

	entity := data[0].(map[string]interface{})
	return entity["id"].(string), nil
}

// GetConfigTypeID retrieves the ID of a config type by name
func (c *ZitiClient) GetConfigTypeID(token, typeName string) (string, error) {
	filter := url.QueryEscape(fmt.Sprintf(`name="%s"`, typeName))
	endpoint := fmt.Sprintf("config-types?filter=%s", filter)

	resp, err := c.request("GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	data := resp["data"].([]interface{})
	if len(data) == 0 {
		return "", fmt.Errorf("config type not found: %s", typeName)
	}

	configType := data[0].(map[string]interface{})
	return configType["id"].(string), nil
}

// CreateIdentity creates a Ziti identity
func (c *ZitiClient) CreateIdentity(name, idType string, attrs []string) (id, jwt string, err error) {
	body := map[string]interface{}{
		"name":  name,
		"type": idType,
		"roleAttributes": attrs,
	}

	resp, err := c.request("POST", "identities", body)
	if err != nil {
		return "", "", err
	}

	data := resp["data"].(map[string]interface{})
	id = data["id"].(string)
	jwt = data["enrollment"].(map[string]interface{})["ott"].(map[string]interface{})["jwt"].(string)

	return id, jwt, nil
}

// CreateEdgeRouter creates an edge router
func (c *ZitiClient) CreateEdgeRouter(name string, attrs []string, tunnelerEnabled bool) (id, jwt string, err error) {
	body := map[string]interface{}{
		"name":             name,
		"roleAttributes":   attrs,
		"tunnelerEnabled":  tunnelerEnabled,
	}

	resp, err := c.request("POST", "edge-routers", body)
	if err != nil {
		return "", "", err
	}

	data := resp["data"].(map[string]interface{})
	id = data["id"].(string)
	jwt = data["enrollment"].(map[string]interface{})["ott"].(map[string]interface{})["jwt"].(string)

	return id, jwt, nil
}

// CreateAuthPolicy creates an auth policy
func (c *ZitiClient) CreateAuthPolicy(name string, data map[string]interface{}) (string, error) {
	data["name"] = name

	resp, err := c.request("POST", "auth-policies", data)
	if err != nil {
		return "", err
	}

	respData := resp["data"].(map[string]interface{})
	return respData["id"].(string), nil
}

// CreateAuthenticator creates an authenticator
func (c *ZitiClient) CreateAuthenticator(identityID, authType string, data map[string]interface{}) (string, error) {
	// Convert to proper format for Ziti
	body := map[string]interface{}{
		"method":     authType,
		"identityId": identityID,
	}

	// Merge additional data
	for k, v := range data {
		body[k] = v
	}

	resp, err := c.request("POST", "authenticators", body)
	if err != nil {
		return "", err
	}

	respData := resp["data"].(map[string]interface{})
	return respData["id"].(string), nil
}

// CreateService creates a Ziti service
func (c *ZitiClient) CreateService(name string, configIDs, attrs []string) (string, error) {
	body := map[string]interface{}{
		"name":              name,
		"roleAttributes":    attrs,
	}

	// Only add configs if provided
	if len(configIDs) > 0 {
		body["configs"] = configIDs
	}

	resp, err := c.request("POST", "services", body)
	if err != nil {
		return "", err
	}

	data := resp["data"].(map[string]interface{})
	return data["id"].(string), nil
}

// CreateConfig creates a Ziti config
func (c *ZitiClient) CreateConfig(name, typeID string, data interface{}) (string, error) {
	body := map[string]interface{}{
		"name":   name,
		"typeId": typeID,
		"data":   data,
	}

	resp, err := c.request("POST", "configs", body)
	if err != nil {
		return "", err
	}

	respData := resp["data"].(map[string]interface{})
	return respData["id"].(string), nil
}

// CreateServicePolicy creates a service policy
func (c *ZitiClient) CreateServicePolicy(name, policyType string, idRoles, svcRoles []string) error {
	body := map[string]interface{}{
		"name":            name,
		"type":            policyType,
		"identityRoles":   idRoles,
		"serviceRoles":    svcRoles,
	}

	_, err := c.request("POST", "service-policies", body)
	return err
}

// CreateEdgeRouterPolicy creates an edge router policy
func (c *ZitiClient) CreateEdgeRouterPolicy(name string, idRoles, routerRoles []string) error {
	body := map[string]interface{}{
		"name":          name,
		"identityRoles": idRoles,
		"edgeRouterRoles": routerRoles,
	}

	_, err := c.request("POST", "edge-router-policies", body)
	return err
}

// CreateServiceEdgeRouterPolicy creates a service edge router policy
func (c *ZitiClient) CreateServiceEdgeRouterPolicy(name string, svcRoles, routerRoles []string) error {
	body := map[string]interface{}{
		"name":          name,
		"serviceRoles": svcRoles,
		"edgeRouterRoles": routerRoles,
	}

	_, err := c.request("POST", "service-edge-router-policies", body)
	return err
}

// ListAuthPolicies lists auth policies
func (c *ZitiClient) ListAuthPolicies() ([]map[string]interface{}, error) {
	resp, err := c.request("GET", "auth-policies?limit=100", nil)
	if err != nil {
		return nil, err
	}

	data := resp["data"].([]interface{})
	var policies []map[string]interface{}
	for _, p := range data {
		policies = append(policies, p.(map[string]interface{}))
	}

	return policies, nil
}

// ListEdgeRouters lists edge routers
func (c *ZitiClient) ListEdgeRouters() ([]map[string]interface{}, error) {
	resp, err := c.request("GET", "edge-routers?limit=100", nil)
	if err != nil {
		return nil, err
	}

	data := resp["data"].([]interface{})
	var routers []map[string]interface{}
	for _, r := range data {
		routers = append(routers, r.(map[string]interface{}))
	}

	return routers, nil
}

// ListIdentities lists identities
func (c *ZitiClient) ListIdentities() ([]map[string]interface{}, error) {
	resp, err := c.request("GET", "identities?limit=100", nil)
	if err != nil {
		return nil, err
	}

	data := resp["data"].([]interface{})
	var identities []map[string]interface{}
	for _, i := range data {
		identities = append(identities, i.(map[string]interface{}))
	}

	return identities, nil
}

// ListServices lists all services
func (c *ZitiClient) ListServices() ([]map[string]interface{}, error) {
	resp, err := c.request("GET", "services?limit=100", nil)
	if err != nil {
		return nil, err
	}

	data := resp["data"].([]interface{})
	var services []map[string]interface{}
	for _, s := range data {
		services = append(services, s.(map[string]interface{}))
	}

	return services, nil
}

// UpdateIdentityAttributes updates role attributes on an identity
func (c *ZitiClient) UpdateIdentityAttributes(identityID string, attrs []string) error {
	// Update identity attributes
	body := map[string]interface{}{
		"roleAttributes": attrs,
	}

	_, err := c.request("PATCH", fmt.Sprintf("identities/%s", identityID), body)
	return err
}

// ExecuteCommand runs a CLI command (for Ziti CLI operations)
func (c *ZitiClient) ExecuteCommand(args ...string) error {
	// This would use the ziti CLI binary
	// Implementation depends on how we shell out to ziti
	return nil
}
