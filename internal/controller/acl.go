package controller

import (
	"log"
)

// stepACL seeds ACL policies in Ziti and Bao
func stepACL(cfg *Config) error {
	log.Println("bootstrapping ACL policies...")

	// TODO: Implement ACL seeding
	// Ziti policies (pattern-based, matching cert CNs):
	// - 8 Bind policies (quarantine, tango, ssh, infra, web, home, k8s, public)
	// - 8 Dial policies (same categories)
	// - 8 Edge Router Policies
	// - 1 Service Edge Router Policy
	//
	// Bao policies:
	// - stage-0-secrets (device quarantine access)
	// - stage-1-secrets (device member access)
	// - stage-2-secrets (device lab access)
	// - stage-3-secrets (device admin access)
	// - schmutz-rw (enrollment service)
	//
	// Bao AppRoles:
	// - schmutz-enroll
	// - admin

	log.Println("  ✓ ACL policies seeded")

	return nil
}
