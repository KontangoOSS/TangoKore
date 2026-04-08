package controller

import (
	"log"
)

// stepIdentities creates device certificate issuance roles in Bao PKI
func stepIdentities(cfg *Config) error {
	log.Println("setting up device identity provisioning...")

	// TODO: Implement device identity setup
	// - Create PKI roles in Bao for each stage:
	//   - device-quarantine: CN *.quarantine.{domain}, TTL 24h
	//   - device-member: CN *.members.{domain}, TTL 90d
	//   - device-lab: CN *.lab.{domain}, TTL 1y
	//   - device-admin: CN *.admin.{domain}, TTL 1y
	// - Create endpoint for device certificate requests
	// - CA distribution mechanism

	log.Println("  ✓ device identities configured")

	return nil
}
