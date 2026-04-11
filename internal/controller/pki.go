package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// stepPKI generates PKI using Bao: root CA → intermediate → all certs
func stepPKI(cfg *Config) error {
	log.Println("step 5/13: configuring PKI with Bao...")

	// Create Bao client with root token from step 4
	rootToken, unsealKey, err := loadBaoInit(cfg)
	if err != nil {
		return fmt.Errorf("load bao init: %w", err)
	}

	// Debug: verify token is loaded
	if rootToken == "" {
		return fmt.Errorf("root token is empty")
	}
	tokenLen := len(rootToken)
	if tokenLen > 20 {
		tokenLen = 20
	}
	log.Printf("  → loaded root token: %s...\n", rootToken[:tokenLen])

	client, err := clients.NewBaoClient("https://127.0.0.1:8200", rootToken, "")
	if err != nil {
		return fmt.Errorf("create bao client: %w", err)
	}

	// 1. Mount PKI engines
	log.Println("  → mounting PKI engines...")
	if err := client.PKIMount("pki"); err != nil {
		return fmt.Errorf("mount pki: %w", err)
	}
	if err := client.PKIMount("pki_int"); err != nil {
		return fmt.Errorf("mount pki_int: %w", err)
	}

	// 2. Configure URLs
	log.Println("  → configuring PKI URLs...")
	baoURL := fmt.Sprintf("https://%s.%s/pki/ca", cfg.Name, cfg.Domain)
	crlURL := fmt.Sprintf("https://%s.%s/pki/crl", cfg.Name, cfg.Domain)
	if err := client.PKIConfigURLs("pki", baoURL, crlURL); err != nil {
		return fmt.Errorf("config urls: %w", err)
	}

	// 3. Generate root CA
	log.Println("  → generating root CA...")
	keyType := "ec"
	if cfg.TestMode {
		keyType = "ec" // Use EC for speed in test mode
	}
	rootCertPEM, err := client.PKIGenerateRoot("pki", keyType, "Kontango Root CA", "87600h")
	if err != nil {
		return fmt.Errorf("generate root: %w", err)
	}

	// 4. Generate intermediate CSR
	log.Println("  → generating intermediate CSR...")
	csrPEM, intKeyPEM, err := client.PKIGenerateIntermediateCSR("pki_int", keyType, "Kontango Intermediate CA")
	if err != nil {
		return fmt.Errorf("generate csr: %w", err)
	}

	// 5. Sign intermediate with root
	log.Println("  → signing intermediate with root CA...")
	signedIntCertPEM, err := client.PKISignIntermediate("pki", csrPEM, "Kontango Intermediate CA", "43800h", 0)
	if err != nil {
		return fmt.Errorf("sign intermediate: %w", err)
	}

	// 6. Set signed intermediate
	log.Println("  → setting signed intermediate...")
	if err := client.PKISetSignedIntermediate("pki_int", signedIntCertPEM); err != nil {
		return fmt.Errorf("set signed intermediate: %w", err)
	}

	// 7. Create PKI role for server certificates
	log.Println("  → creating server certificate role...")
	if err := client.CreatePKIRoleOnMount("pki_int", "server",
		[]string{cfg.Domain, "*."+cfg.Domain, "*.tango"}, "8760h"); err != nil {
		return fmt.Errorf("create server role: %w", err)
	}

	// 8. Write certs to disk
	log.Println("  → writing certificates to disk...")
	pems := map[string]string{
		"root-ca.crt":     rootCertPEM,
		"intermediate.crt": signedIntCertPEM,
		"intermediate.key": intKeyPEM,
	}

	for name, pem := range pems {
		path := filepath.Join(cfg.EtcDir, "pki", name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		mode := os.FileMode(0644)
		if strings.Contains(name, "key") {
			mode = 0600
		}
		if err := os.WriteFile(path, []byte(pem), mode); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	// 8. Create signing chain
	signingChain := signedIntCertPEM + "\n" + rootCertPEM
	if err := os.WriteFile(filepath.Join(cfg.EtcDir, "pki", "signing-chain.crt"), []byte(signingChain), 0644); err != nil {
		return fmt.Errorf("write signing chain: %w", err)
	}

	// 9. Create CA bundle (same as signing chain)
	if err := os.WriteFile(filepath.Join(cfg.EtcDir, "pki", "ca-bundle.pem"), []byte(signingChain), 0644); err != nil {
		return fmt.Errorf("write ca bundle: %w", err)
	}

	// 10. Issue server cert for this node (will be used for Bao TLS)
	log.Println("  → issuing server certificate...")
	serverCert, serverKey, _, err := client.PKIIssueCert("pki_int", "server", cfg.Name+"."+cfg.Domain, "8760h", []string{cfg.Domain})
	if err != nil {
		return fmt.Errorf("issue server cert: %w", err)
	}

	if err := os.WriteFile(filepath.Join(cfg.EtcDir, "pki", "server.crt"), []byte(serverCert), 0644); err != nil {
		return fmt.Errorf("write server cert: %w", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.EtcDir, "pki", "server.key"), []byte(serverKey), 0600); err != nil {
		return fmt.Errorf("write server key: %w", err)
	}

	// 11. Issue Bao's own TLS cert (for bao.tango internal DNS)
	log.Println("  → issuing Bao TLS certificate...")
	baoCert, baoKey, _, err := client.PKIIssueCert("pki_int", "server", "bao.tango", "8760h", nil)
	if err != nil {
		return fmt.Errorf("issue bao cert: %w", err)
	}

	if err := os.WriteFile(filepath.Join(cfg.EtcDir, "pki", "bao-server.crt"), []byte(baoCert), 0644); err != nil {
		return fmt.Errorf("write bao cert: %w", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.EtcDir, "pki", "bao-server.key"), []byte(baoKey), 0600); err != nil {
		return fmt.Errorf("write bao key: %w", err)
	}

	// 12. Create PKI role definitions for device certificates (to be created in step 10)
	log.Println("  → defining PKI roles...")
	if err := definePKIRoles(); err != nil {
		return fmt.Errorf("define roles: %w", err)
	}

	// 13. Save unseal key for reference (will be in Bao KV after step 6)
	cfg.BaoUnsealKey = unsealKey
	cfg.BaoRootToken = rootToken

	log.Println("  ✓ PKI configured with Bao root CA")
	return nil
}

// loadBaoInit loads the temporary bao-init.json from step 4
func loadBaoInit(cfg *Config) (rootToken, unsealKey string, err error) {
	initPath := filepath.Join(cfg.EtcDir, "bao-init.json")
	data, err := os.ReadFile(initPath)
	if err != nil {
		return "", "", fmt.Errorf("read init: %w", err)
	}

	var init map[string]string
	if err := json.Unmarshal(data, &init); err != nil {
		return "", "", fmt.Errorf("unmarshal init: %w", err)
	}

	return init["root_token"], init["unseal_key"], nil
}

// definePKIRoles creates role definitions for device certificates
// These will be created in Bao during step 10 (stepIdentities)
func definePKIRoles() error {
	roles := map[string]map[string]interface{}{
		"device-base": {
			"allowed_domains":  []string{"web.example.com"},
			"allow_subdomains": true,
			"max_ttl":          "8760h",
			"key_type":         "ec",
			"description":      "Base device identity (1 year)",
		},
		"device-web": {
			"allowed_domains":  []string{"web.example.com"},
			"allow_subdomains": true,
			"max_ttl":          "24h",
			"key_type":         "ec",
			"description":      "Web testing (24h, quarantine)",
		},
		"device-temp": {
			"allowed_domains":  []string{"temp.example.com"},
			"allow_subdomains": true,
			"max_ttl":          "168h",
			"key_type":         "ec",
			"description":      "Temporary (1 week)",
		},
		"device-tango": {
			"allowed_domains":  []string{"tango"},
			"allow_subdomains": true,
			"max_ttl":          "720h",
			"key_type":         "ec",
			"description":      "Overlay network (30 days)",
		},
	}

	// Store role definitions for later use in step 10
	// For now, just log that they're defined
	for name := range roles {
		log.Printf("    - %s\n", name)
	}

	return nil
}
