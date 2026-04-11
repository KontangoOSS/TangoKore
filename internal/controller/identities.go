package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// stepIdentities creates device certificate issuance roles in Bao PKI and registers Ziti CA
func stepIdentities(cfg *Config) error {
	log.Println("step 10/13: configuring device identities...")

	// Skip on edge routers — only controllers register CA and create roles
	if cfg.JoinMode {
		log.Println("  ⚠ skipping identities (edge router)")
		return nil
	}

	// Load Bao root token from step 4
	rootToken, _, err := loadBaoInit(cfg)
	if err != nil {
		return fmt.Errorf("load bao init: %w", err)
	}

	baoClient, err := clients.NewBaoClient("https://127.0.0.1:8200", rootToken, "")
	if err != nil {
		return fmt.Errorf("create bao client: %w", err)
	}

	// 1. Create PKI roles in pki_int mount
	log.Println("  → creating PKI roles...")
	roles := getPKIRoles(cfg)
	for roleName, roleConfig := range roles {
		domains := roleConfig["allowed_domains"].([]string)
		maxTTL := roleConfig["max_ttl"].(string)

		// Create role on pki_int mount
		if err := createPKIRoleOnMount(baoClient, "pki_int", roleName, domains, maxTTL); err != nil {
			return fmt.Errorf("create role %s: %w", roleName, err)
		}
		desc := roleConfig["description"].(string)
		log.Printf("    ✓ %s (TTL: %s) — %s", roleName, maxTTL, desc)
	}

	// 2. Register Ziti 3rd party CA
	log.Println("  → registering Ziti CA...")

	// Load intermediate cert
	intCertPath := filepath.Join(cfg.EtcDir, "pki", "intermediate.crt")
	intCert, err := os.ReadFile(intCertPath)
	if err != nil {
		return fmt.Errorf("read int cert: %w", err)
	}

	// Create Ziti client
	zitiClient, err := clients.NewZitiClient(
		fmt.Sprintf("https://127.0.0.1:%d/edge/management/v1", cfg.ZitiCtrlPort),
		cfg.ZitiAdminUser,
		cfg.ZitiAdminPass,
	)
	if err != nil {
		return fmt.Errorf("create ziti client: %w", err)
	}

	// Register CA: autoCa=true, ottCa=true (one-time-token CA), authEnabled=true
	caID, verificationToken, err := zitiClient.CreateCA("bao-device-ca", string(intCert), true, true, true)
	if err != nil {
		return fmt.Errorf("create ziti ca: %w", err)
	}
	log.Printf("    → CA created (ID: %s, needs verification)", caID)

	// 3. Verify CA by issuing a verification cert
	log.Println("  → verifying CA...")

	// Issue verification cert: CN = verificationToken, 1 hour TTL, from pki_int
	verifyCert, _, _, err := baoClient.PKIIssueCert(
		"pki_int",
		"device-base",
		verificationToken,
		"1h",
		nil,
	)
	if err != nil {
		return fmt.Errorf("issue verify cert: %w", err)
	}

	// Verify CA with the issued cert
	if err := zitiClient.VerifyCA(caID, verifyCert); err != nil {
		return fmt.Errorf("verify ca: %w", err)
	}

	log.Printf("    ✓ CA verified and active")

	log.Println("  ✓ device identities configured")
	return nil
}

// getPKIRoles returns the device certificate issuance roles
func getPKIRoles(cfg *Config) map[string]map[string]interface{} {
	webDomain := fmt.Sprintf("web.%s", cfg.Domain)

	return map[string]map[string]interface{}{
		"device-base": {
			"allowed_domains": []string{webDomain},
			"max_ttl":         "8760h",
			"description":     "Base device identity (1 year)",
		},
		"device-web": {
			"allowed_domains": []string{webDomain},
			"max_ttl":         "24h",
			"description":     "Web testing (24h, quarantine)",
		},
		"device-temp": {
			"allowed_domains": []string{fmt.Sprintf("temp.%s", cfg.Domain)},
			"max_ttl":         "168h",
			"description":     "Temporary (1 week)",
		},
		"device-lab": {
			"allowed_domains": []string{fmt.Sprintf("lab.%s", cfg.Domain)},
			"max_ttl":         "4380h",
			"description":     "Lab (6 months)",
		},
		"device-admin": {
			"allowed_domains": []string{webDomain},
			"max_ttl":         "87600h",
			"description":     "Admin (10 years)",
		},
	}
}

// createPKIRoleOnMount creates a PKI role on a specific mount
func createPKIRoleOnMount(c *clients.BaoClient, mountPath, roleName string, allowedDomains []string, maxTTL string) error {
	return c.CreatePKIRoleOnMount(mountPath, roleName, allowedDomains, maxTTL)
}
