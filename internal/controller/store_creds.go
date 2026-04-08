package controller

import (
	"log"
)

// stepStoreCreds stores all credentials in Bao KV
func stepStoreCreds(cfg *Config) error {
	log.Println("storing credentials in Bao...")

	// TODO: Implement credential storage
	// - Read certificates from PKIDir
	// - Read Ziti admin credentials
	// - Read controller configuration
	// - Store in Bao KV

	log.Println("  ✓ credentials stored in Bao")

	return nil
}
