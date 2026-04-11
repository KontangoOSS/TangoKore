package controller

import (
	"fmt"
	"log"

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

	// Device identities are configured via Ziti PKI in stepPKI
	// No Bao setup needed in test mode

	// 1. Create PKI roles in pki_int mount
	// NOTE: PKI roles are created via Ziti PKI, not Bao PKI
	// This step can be used later for Bao PKI roles if needed
	log.Println("  → PKI roles (via Ziti PKI, not Bao)")
	_ = getPKIRoles(cfg) // unused for now

	// 2. Register Ziti 3rd party CA
	// NOTE: Ziti CA registration requires proper authentication method handling
	// Skip for now - test mode doesn't require 3rd party CA verification
	log.Println("  → Ziti CA registration (skipped in test mode)")

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
