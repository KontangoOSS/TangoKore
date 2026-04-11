package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config holds all configuration for controller installation
type Config struct {
	// Node identity
	Name          string // e.g. "ctrl-1"
	Domain        string // e.g. "konoss.org"
	JoinDomain    string // e.g. "ctrl.konoss.org"
	OverlayDomain string // e.g. "tango"

	// Mode
	JoinMode         bool
	JoinLeader       string // "root@ctrl-1.tango"
	JoinBaoUnsealKey string

	// Credentials
	ZitiAdminUser   string
	ZitiAdminPass   string
	CloudflareToken string
	ACMEEmail       string

	// Test mode
	TestMode bool

	// Versions
	ZitiVersion string
	BaoVersion  string

	// Paths
	Home   string
	EtcDir string

	// Derived — set by Install() before first step
	BinDir  string
	PKIDir  string
	ZitiDir string
	DataDir string

	// Ports
	ZitiCtrlPort int
	ZitiEdgePort int
	ZitiLinkPort int
	BaoPort      int
	BaoRaftPort  int
	SchmutzPort  int

	// Accumulated during install
	BaoUnsealKey string
	BaoRootToken string
	NodePublicIP string
}

type step struct {
	name string
	fn   func(*Config) error
	skip func(*Config) bool // Return true to skip this step
}

// buildSteps constructs the step list with conditional skip logic
func buildSteps(cfg *Config) []step {
	return []step{
		{"preflight", stepPreflight, nil},
		{"download", stepDownload, nil},
		// Bao: init for leader, join for followers
		{"bao-init", stepBaoInit, func(c *Config) bool { return c.JoinMode }},
		{"bao-join", stepBaoJoin, func(c *Config) bool { return !c.JoinMode }},
		// PKI: generate for leader, fetch from leader for followers
		{"pki", stepPKI, func(c *Config) bool { return c.JoinMode }},
		{"pki-from-leader", stepPKIFromLeader, func(c *Config) bool { return !c.JoinMode }},
		{"ziti", stepZiti, nil},
		// Store credentials: controller-only (not replicated from leader)
		{"store-creds", stepStoreCreds, func(c *Config) bool { return c.JoinMode }},
		// Caddy: skipped (controllers don't need Caddy)
		{"caddy", stepCaddy, func(c *Config) bool { return true }},
		{"schmutz", stepSchmutz, nil},
		// Controller-only: identities, fabric, acl
		{"identities", stepIdentities, func(c *Config) bool { return c.JoinMode }},
		{"fabric", stepFabric, func(c *Config) bool { return c.JoinMode }},
		{"acl", stepACL, func(c *Config) bool { return c.JoinMode }},
		{"verify", stepVerify, nil},
	}
}

// Install runs the complete controller bootstrap
func Install(cfg *Config) error {
	log.SetFlags(0)

	// Set derived paths
	cfg.BinDir = filepath.Join(cfg.Home, "bin")
	cfg.PKIDir = filepath.Join(cfg.Home, "pki")
	cfg.ZitiDir = filepath.Join(cfg.Home, "ziti")
	cfg.DataDir = filepath.Join(cfg.Home, "data", "bao")

	// Set default ports if not specified
	if cfg.ZitiCtrlPort == 0 {
		cfg.ZitiCtrlPort = 1280
	}
	if cfg.ZitiEdgePort == 0 {
		cfg.ZitiEdgePort = 3023
	}
	if cfg.ZitiLinkPort == 0 {
		cfg.ZitiLinkPort = 3022
	}
	if cfg.BaoPort == 0 {
		cfg.BaoPort = 8200
	}
	if cfg.BaoRaftPort == 0 {
		cfg.BaoRaftPort = 8201
	}
	if cfg.SchmutzPort == 0 {
		cfg.SchmutzPort = 3080
	}

	// Set default versions
	if cfg.ZitiVersion == "" {
		cfg.ZitiVersion = "1.6.15"
	}
	if cfg.BaoVersion == "" {
		cfg.BaoVersion = "2.5.2"
	}

	// Set default domain if in test mode
	if cfg.TestMode && cfg.Domain == "" {
		cfg.Domain = "kontango.local"
	}

	// Set default overlay domain
	if cfg.OverlayDomain == "" {
		cfg.OverlayDomain = "tango"
	}

	// Set default join domain
	if cfg.JoinDomain == "" {
		cfg.JoinDomain = "join." + cfg.Domain
	}

	log.Printf("\n========================================")
	log.Printf("kontango controller install")
	log.Printf("========================================\n")

	if cfg.TestMode {
		log.Printf("MODE: test (self-signed certs)\n")
	} else {
		log.Printf("MODE: production (ACME/Cloudflare)\n")
	}

	if cfg.JoinMode {
		log.Printf("TYPE: join mode (replicating from %s)\n", cfg.JoinLeader)
	} else {
		log.Printf("TYPE: init mode (bootstrap cluster)\n")
	}

	log.Printf("NODE: %s\n", cfg.Name)
	log.Printf("DOMAIN: %s\n", cfg.Domain)
	log.Printf("HOME: %s\n", cfg.Home)
	log.Printf("OVERLAY: %s\n\n", cfg.OverlayDomain)

	// Build step list with conditional skip logic
	steps := buildSteps(cfg)

	// Run each step
	for _, s := range steps {
		// Check if step should be skipped
		if s.skip != nil && s.skip(cfg) {
			log.Printf("[%s] skipped (conditional)\n", s.name)
			continue
		}

		// Check if already completed
		sentinel := filepath.Join(cfg.EtcDir, fmt.Sprintf(".step-%s-done", s.name))
		if _, err := os.Stat(sentinel); err == nil {
			log.Printf("[%s] already completed, skipping\n", s.name)
			continue
		}

		log.Printf("\n--- step: %s ---\n", s.name)

		if err := s.fn(cfg); err != nil {
			log.Printf("[%s] FAILED: %v\n", s.name, err)
			return fmt.Errorf("%s: %w", s.name, err)
		}

		// Write sentinel
		if err := os.MkdirAll(cfg.EtcDir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", cfg.EtcDir, err)
		}
		if err := os.WriteFile(sentinel, []byte(""), 0644); err != nil {
			return fmt.Errorf("write sentinel: %w", err)
		}

		log.Printf("[%s] OK\n", s.name)
	}

	log.Printf("\n========================================")
	log.Printf("kontango controller bootstrap complete")
	log.Printf("========================================\n\n")

	return nil
}

