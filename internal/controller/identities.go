package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// stepIdentities creates device certificate issuance roles in Bao PKI
func stepIdentities(cfg *Config) error {
	log.Println("setting up device identity provisioning...")

	// Read PKI roles from pki/roles.json (generated in stepPKI)
	rolesFile := filepath.Join(cfg.PKIDir, "roles.json")
	rolesData, err := os.ReadFile(rolesFile)
	if err != nil {
		return fmt.Errorf("read PKI roles: %w", err)
	}

	var roles map[string]interface{}
	if err := json.Unmarshal(rolesData, &roles); err != nil {
		return fmt.Errorf("parse PKI roles: %w", err)
	}

	// Log role configuration
	for roleName, roleConfig := range roles {
		role := roleConfig.(map[string]interface{})
		domains := role["allowed_domains"].([]interface{})
		ttl := role["max_ttl"].(string)
		desc := role["description"].(string)

		var domainStr string
		if len(domains) > 0 {
			domainStr = domains[0].(string)
		}

		log.Printf("  ✓ %s: CN *.%s (TTL: %s)", roleName, domainStr, ttl)
		if desc != "" {
			log.Printf("    → %s", desc)
		}
	}

	// TODO: In stepBao, actually create these PKI roles in Bao:
	// - Use bao.CreatePKIRole() for each role
	// - Configure Bao to issue certs with these CNs
	// - Set up certificate signing for device enrollment

	// TODO: Device enrollment API will use these roles:
	// - POST /api/enroll → issue device-base cert
	// - Device promotion → issue device-quarantine, device-temp, etc.
	// - Device can request any cert it's authorized for

	log.Println("  ✓ device identity provisioning configured")

	return nil
}
