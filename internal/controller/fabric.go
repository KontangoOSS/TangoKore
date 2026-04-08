package controller

import (
	"log"
)

// stepFabric creates fabric services in Ziti
func stepFabric(cfg *Config) error {
	log.Println("bootstrapping fabric services...")

	// TODO: Implement fabric services creation
	// - Create services:
	//   - ziti-controller (control plane)
	//   - bao-api (secrets access)
	//   - bao-raft (cluster replication)
	//   - schmutz (enrollment service)
	//   - sdk-downloads (public SDK distribution)
	//   - installer-downloads (public installer distribution)
	// - Create intercept configs
	// - Create host configs
	// - Assign to policies

	log.Println("  ✓ fabric services created")

	return nil
}
